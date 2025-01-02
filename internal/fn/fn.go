// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package fn

import (
	"context"
	"fmt"
	"log"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/parser"

	input "github.com/crossplane-contrib/function-cue/input/v1beta1"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/function-sdk-go"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/response"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

const debugAnnotation = "cue.fn.crossplane.io/debug"

// Options are options for the cue runner.
type Options struct {
	Logger logging.Logger
	Debug  bool
}

// Cue runs cue scripts that adhere to a specific interface.
type Cue struct {
	fnv1.UnimplementedFunctionRunnerServiceServer
	log   logging.Logger
	debug bool
}

// New creates a cue runner.
func New(opts Options) (*Cue, error) {
	if opts.Logger == nil {
		var err error
		opts.Logger, err = function.NewLogger(opts.Debug)
		if err != nil {
			return nil, err
		}
	}
	return &Cue{
		log:   opts.Logger,
		debug: opts.Debug,
	}, nil
}

// DebugOptions are per-eval debug options.
type DebugOptions struct {
	Enabled bool // enable input/ output debugging
	Raw     bool // do not remove any "noise" attributes in the input object
	Script  bool // render the final script as a debug output
}

type EvalOptions struct {
	RequestVar          string
	ResponseVar         string
	DesiredOnlyResponse bool
	Debug               DebugOptions
}

// Eval evaluates the supplied script with an additional script that includes the supplied request and returns the
// response.
func (f *Cue) Eval(in *fnv1.RunFunctionRequest, script string, opts EvalOptions) (*fnv1.RunFunctionResponse, error) {
	// input request only contains properties as documented in the interface, not the whole object
	req := &fnv1.RunFunctionRequest{
		Observed: in.GetObserved(),
		Desired:  in.GetDesired(),
		Context:  in.GetContext(),
	}
	// extract request as object
	reqBytes, err := protojson.MarshalOptions{Indent: "  "}.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(err, "proto json marshal")
	}

	preamble := fmt.Sprintf("%s: ", opts.RequestVar)
	if opts.Debug.Enabled {
		log.Printf("[request:begin]\n%s %s\n[request:end]\n", preamble, f.getDebugString(reqBytes, opts.Debug.Raw))
	}

	finalScript := fmt.Sprintf("%s\n%s %s\n", script, preamble, reqBytes)
	if opts.Debug.Script {
		log.Printf("[script:begin]\n%s\n[script:end]\n", finalScript)
	}

	runtime := cuecontext.New()
	val := runtime.CompileBytes([]byte(finalScript))
	if val.Err() != nil {
		return nil, errors.Wrap(val.Err(), "compile cue code")
	}

	if opts.ResponseVar != "" {
		e, err := parser.ParseExpr("expression", opts.ResponseVar)
		if err != nil {
			return nil, errors.Wrap(err, "parse response expression")
		}
		val = val.Context().BuildExpr(e,
			cue.Scope(val),
			cue.InferBuiltins(true),
		)
		if val.Err() != nil {
			return nil, errors.Wrap(val.Err(), "build response expression")
		}
	}

	resBytes, err := val.MarshalJSON() // this can fail if value is not concrete
	if err != nil {
		return nil, errors.Wrap(err, "marshal cue output")
	}
	if opts.Debug.Enabled {
		preamble = ""
		if opts.ResponseVar != "" {
			preamble = opts.ResponseVar + ":"
		}
		log.Printf("[response:begin]\n%s %s\n[response:end]\n", preamble, f.getDebugString(resBytes, opts.Debug.Raw))
	}

	var ret fnv1.RunFunctionResponse
	if opts.DesiredOnlyResponse {
		var state fnv1.State
		err = protojson.Unmarshal(resBytes, &state)
		if err == nil {
			ret.Desired = &state
		}
	} else {
		err = protojson.Unmarshal(resBytes, &ret)
	}
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal cue output using proto json")
	}
	return &ret, nil
}

// RunFunction runs the function. It expects a single script that is complete, except for a request
// variable that the function runner supplies.
func (f *Cue) RunFunction(_ context.Context, req *fnv1.RunFunctionRequest) (outRes *fnv1.RunFunctionResponse, finalErr error) {
	// setup response with desired state set up upstream functions
	res := response.To(req, response.DefaultTTL)

	logger := f.log
	// automatically handle errors and response logging
	defer func() {
		if finalErr == nil {
			logger.Info("cue module executed successfully")
			response.Normal(outRes, "cue module executed successfully")
			return
		}
		logger.Info(finalErr.Error())
		response.Fatal(res, finalErr)
		outRes = res
	}()

	// setup logging and debugging
	oxr, err := request.GetObservedCompositeResource(req)
	if err != nil {
		return nil, errors.Wrap(err, "get observed composite")
	}
	tag := req.GetMeta().GetTag()
	if tag != "" {
		logger = f.log.WithValues("tag", tag)
	}
	logger = logger.WithValues(
		"xr-version", oxr.Resource.GetAPIVersion(),
		"xr-kind", oxr.Resource.GetKind(),
		"xr-name", oxr.Resource.GetName(),
	)
	logger.Info("Running Function")
	debugThis := false
	annotations := oxr.Resource.GetAnnotations()
	if annotations != nil && annotations[debugAnnotation] == "true" {
		debugThis = true
	}

	// get inputs
	in := &input.CueInput{}
	if err := request.GetInput(req, in); err != nil {
		return nil, errors.Wrap(err, "unable to get input")
	}
	if in.Script == "" {
		return nil, fmt.Errorf("input script was not specified")
	}
	if in.DebugNew {
		if len(req.GetObserved().GetResources()) == 0 {
			debugThis = true
		}
	}
	// set up the request and response variables
	requestVar := "#request"
	if in.RequestVar != "" {
		requestVar = in.RequestVar
	}
	var responseVar string
	switch in.ResponseVar {
	case ".":
		responseVar = ""
	case "":
		responseVar = "response"
	default:
		responseVar = in.ResponseVar
	}
	state, err := f.Eval(req, in.Script, EvalOptions{
		RequestVar:          requestVar,
		ResponseVar:         responseVar,
		DesiredOnlyResponse: in.LegacyDesiredOnlyResponse,
		Debug: DebugOptions{
			Enabled: f.debug || in.Debug || debugThis,
			Raw:     in.DebugRaw,
			Script:  in.DebugScript,
		},
	})
	if err != nil {
		return res, errors.Wrap(err, "eval script")
	}
	return f.mergeResponse(res, state)
}

func (f *Cue) mergeResponse(res *fnv1.RunFunctionResponse, cueResponse *fnv1.RunFunctionResponse) (*fnv1.RunFunctionResponse, error) {
	// selectively add returned resources without deleting any previous desired state
	if res.Desired == nil {
		res.Desired = &fnv1.State{}
	}
	if res.Desired.Resources == nil {
		res.Desired.Resources = map[string]*fnv1.Resource{}
	}
	// only set desired composite if the cue script actually returns it
	// TODO: maybe use fieldpath.Pave to only extract status
	if cueResponse.Desired.GetComposite() != nil {
		res.Desired.Composite = cueResponse.Desired.GetComposite()
	}
	// set desired resources from cue output
	for k, v := range cueResponse.Desired.GetResources() {
		res.Desired.Resources[k] = v
	}
	// merge the context if cueResponse has something in it
	if cueResponse.Context != nil {
		ctxMap := map[string]interface{}{}
		// set up base map, if found
		if res.Context != nil {
			ctxMap = res.Context.AsMap()
		}
		// merge values from cueResponse
		for k, v := range cueResponse.Context.AsMap() {
			ctxMap[k] = v
		}
		s, err := structpb.NewStruct(ctxMap)
		if err != nil {
			return nil, errors.Wrap(err, "set response context")
		}
		res.Context = s
	}
	// TODO: allow the cue layer to set warnings in cueResponse?
	return res, nil
}
