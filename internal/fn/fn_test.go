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
	"encoding/json"
	"strings"
	"testing"

	input "github.com/crossplane-contrib/function-cue/input/v1beta1"
	"google.golang.org/protobuf/types/known/structpb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fnv1beta1 "github.com/crossplane/function-sdk-go/proto/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

func makeRequest(t *testing.T) *fnv1beta1.RunFunctionRequest {
	var req fnv1beta1.RunFunctionRequest
	reqJSON := `
{
	"meta": { "tag": "v1" },
    "observed": {
        "composite": {
            "resource": {
				"apiVersion": "v1",
				"kind": "MyKind",
				"metadata": {
					"annotations": { "crossplane-function-cue/debug": "true" }
				},
				"foo": "bar" 
			},
			"ready": 0
        }
    },
	"input": {
		"foo": "bar"
	}
}
`
	err := protojson.Unmarshal([]byte(reqJSON), &req)
	require.NoError(t, err)
	return &req
}

func TestEval(t *testing.T) {
	script := `
package runtime
#request: {...}
response: desired: resources: main: resource: {
	foo: #request.observed.composite.resource.foo
	bar: "baz"
}
`
	f, err := New(Options{})
	require.NoError(t, err)
	req := makeRequest(t)
	res, err := f.Eval(req, script, EvalOptions{
		RequestVar:  "#request",
		ResponseVar: "response",
		Debug:       DebugOptions{Enabled: true, Script: true},
	})
	require.NoError(t, err)
	b, _ := protojson.Marshal(res)
	blanksRemoved := strings.ReplaceAll(string(b), " ", "")
	assert.Equal(t, `{"desired":{"resources":{"main":{"resource":{"bar":"baz","foo":"bar"}}}}}`, blanksRemoved)
}

func TestEvalLegacy(t *testing.T) {
	script := `
package runtime
_request: {...}
resources: main: resource: {
	foo: _request.observed.composite.resource.foo
	bar: "baz"
}
`
	f, err := New(Options{})
	require.NoError(t, err)
	req := makeRequest(t)
	res, err := f.Eval(req, script, EvalOptions{
		RequestVar:          "_request",
		ResponseVar:         "",
		DesiredOnlyResponse: true,
		Debug:               DebugOptions{Enabled: true, Script: true},
	})
	require.NoError(t, err)
	b, _ := protojson.Marshal(res)
	blanksRemoved := strings.ReplaceAll(string(b), " ", "")
	assert.Equal(t, `{"desired":{"resources":{"main":{"resource":{"bar":"baz","foo":"bar"}}}}}`, blanksRemoved)
}

func TestEvalBadRuntimeCode(t *testing.T) {
	script := `
package runtime
request: {...}
response: desired: resources: main: resource: {
	foo: request.observed.composite.resource.NO_SUCH_FIELD
	bar: "baz"
}
`
	f, err := New(Options{})
	require.NoError(t, err)
	req := makeRequest(t)
	_, err = f.Eval(req, script, EvalOptions{
		RequestVar:  "request",
		ResponseVar: "response",
		Debug:       DebugOptions{Enabled: true, Script: true},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "undefined field: NO_SUCH_FIELD")
}

func TestEvalBadSourceCode(t *testing.T) {
	script := `
package runtime
request: {...}
response: desired: resources: main: resource: { // no closing brace
`
	f, err := New(Options{})
	require.NoError(t, err)
	req := makeRequest(t)
	_, err = f.Eval(req, script, EvalOptions{
		RequestVar:  "request",
		ResponseVar: "response",
		Debug:       DebugOptions{Enabled: true, Script: true},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "compile cue code: expected '}', found 'EOF'")
}

func TestEvalBadReturnState(t *testing.T) {
	script := `
package runtime
response: desired: foo: "bar" // output does not conform to the State message
`
	f, err := New(Options{})
	require.NoError(t, err)
	req := makeRequest(t)
	_, err = f.Eval(req, script, EvalOptions{
		RequestVar:  "request",
		ResponseVar: "response",
		Debug:       DebugOptions{Enabled: true, Script: true},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal cue output using proto json")
	assert.Contains(t, err.Error(), `unknown field "foo"`)
}

func TestMergeResponse(t *testing.T) {
	responseJSON := `
{
	"desired":	{
		"resources": {
			"main": {
				"resource": { "foo": "bar" },
				"ready": 1
			}
		},
		"composite": {
			"resource": { "foo": "bar" },
			"ready": 1
		}
	},
	"context": {
		"foo": "bar"
	}
}
`
	var cueRes fnv1beta1.RunFunctionResponse
	err := protojson.Unmarshal([]byte(responseJSON), &cueRes)
	require.NoError(t, err)
	var res fnv1beta1.RunFunctionResponse
	f, err := New(Options{})
	require.NoError(t, err)
	_, err = f.mergeResponse(&res, &cueRes)
	require.NoError(t, err)
	b, _ := protojson.Marshal(&res)
	blanksRemoved := strings.ReplaceAll(string(b), " ", "")
	assert.Equal(t, `{"desired":{"composite":{"resource":{"foo":"bar"},"ready":"READY_TRUE"},"resources":{"main":{"resource":{"foo":"bar"},"ready":"READY_TRUE"}}},"context":{"foo":"bar"}}`, blanksRemoved)
}

func TestMergeResponseWithExisting(t *testing.T) {
	existingJSON := `
{
	"desired":	{
		"resources": {
			"supplementary": {
				"resource": { "foo": "bar" },
				"ready": 1
			}
		}
	},
	"context": {
		"foo": "foo2",
		"bar": "baz"
	}
}
`
	responseJSON := `
{
	"desired":	{
		"resources": {
			"main": {
				"resource": { "foo": "bar" },
				"ready": 1
			}
		},
		"composite": {
			"resource": { "foo": "bar" },
			"ready": 1
		}
	},
	"context": {
		"foo": "bar"
	}
}
`
	var cueRes fnv1beta1.RunFunctionResponse
	err := protojson.Unmarshal([]byte(responseJSON), &cueRes)
	require.NoError(t, err)
	var res fnv1beta1.RunFunctionResponse
	err = protojson.Unmarshal([]byte(existingJSON), &res)
	require.NoError(t, err)
	f, err := New(Options{})
	require.NoError(t, err)
	_, err = f.mergeResponse(&res, &cueRes)
	require.NoError(t, err)
	b, _ := protojson.Marshal(&res)
	blanksRemoved := strings.ReplaceAll(string(b), " ", "")
	assert.Equal(t, `{"desired":{"composite":{"resource":{"foo":"bar"},"ready":"READY_TRUE"},"resources":{"main":{"resource":{"foo":"bar"},"ready":"READY_TRUE"},"supplementary":{"resource":{"foo":"bar"},"ready":"READY_TRUE"}}},"context":{"bar":"baz","foo":"bar"}}`, blanksRemoved)
}

func TestRunFunction(t *testing.T) {
	req := makeRequest(t)
	script := `
package runtime
#request: {...}
response: desired: resources: main: resource: {
	foo: #request.observed.composite.resource.foo
	bar: "baz"
}
`
	in := input.CueInput{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1Aplha1", Kind: "Function"},
		ObjectMeta: metav1.ObjectMeta{Name: "foobar"},
		Script:     script,
		Debug:      true,
	}
	b, err := json.Marshal(in)
	require.NoError(t, err)
	var untyped structpb.Struct
	err = protojson.Unmarshal(b, &untyped)
	require.NoError(t, err)
	req.Input = &untyped

	f, err := New(Options{Debug: true})
	require.NoError(t, err)
	res, err := f.RunFunction(context.Background(), req)
	require.NoError(t, err)
	b, err = protojson.Marshal(res)
	require.NoError(t, err)
	blanksRemoved := strings.ReplaceAll(string(b), " ", "")
	assert.Equal(t, `{"meta":{"tag":"v1","ttl":"60s"},"desired":{"resources":{"main":{"resource":{"bar":"baz","foo":"bar"}}}},"results":[{"severity":"SEVERITY_NORMAL","message":"cuemoduleexecutedsuccessfully"}]}`, blanksRemoved)
}
