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

package cuetools

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/cue/parser"
	"github.com/crossplane-contrib/function-cue/internal/fn"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var TestOutput io.Writer = os.Stderr

type TestConfig struct {
	Package                   string
	TestPackage               string
	TestTags                  []string
	RequestVar                string
	ResponseVar               string
	LegacyDesiredOnlyResponse bool
	Debug                     bool
}

type Tester struct {
	config *TestConfig
}

type assertionMode string

const (
	AssertionModeDiff        assertionMode = "diff"
	AssertionModeUnification assertionMode = "unification"
)

type ErrUnknownAssertionMode struct {
	Mode string
}

func (e *ErrUnknownAssertionMode) Error() string {
	return fmt.Sprintf("unknown assertion mode: %s", e.Mode)
}

func assertionModeFromString(mode string) (assertionMode, error) {
	switch mode {
	case "diff":
		return AssertionModeDiff, nil
	case "unification":
		return AssertionModeUnification, nil
	default:
		return "", &ErrUnknownAssertionMode{Mode: mode}
	}
}

// NewTester returns a test for the supplied configuration. It auto-discovers tags from test file names if needed.
func NewTester(config TestConfig) (*Tester, error) {
	ret := &Tester{config: &config}
	if err := ret.init(); err != nil {
		return nil, err
	}
	return ret, nil
}

func (t *Tester) init() error {
	if t.config.Package == "" {
		return fmt.Errorf("package was not specified")
	}
	if t.config.TestPackage == "" {
		t.config.TestPackage = fmt.Sprintf("%s/%s", strings.TrimSuffix(t.config.Package, "/"), "tests")
	}
	// discover test tags from filenames
	if len(t.config.TestTags) == 0 {
		err := t.discoverTags()
		if err != nil {
			return errors.Wrap(err, "discover tags")
		}
		sort.Strings(t.config.TestTags)
	}
	if len(t.config.TestTags) == 0 {
		return fmt.Errorf("no test tags found even after auto-discovery")
	}
	_, _ = fmt.Fprintf(TestOutput, "running test tags: %s\n", strings.Join(t.config.TestTags, ", "))
	return nil
}

func (t *Tester) discoverTags() error {
	pattern := fmt.Sprintf("%s/*.cue", strings.TrimSuffix(t.config.TestPackage, "/"))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return errors.Wrapf(err, "glob %s", pattern)
	}
	for _, name := range matches {
		base := filepath.Base(name)
		pos := strings.Index(base, ".")
		tag := base
		if pos > 0 {
			tag = base[:pos]
		}
		t.config.TestTags = append(t.config.TestTags, tag)
	}
	return nil
}

func evalPackage(pkg string, tag string, expr string, into proto.Message) (cue.Value, error) {
	iv, err := loadSingleInstanceValue(pkg, &load.Config{Tags: []string{tag}})
	if err != nil {
		return cue.Value{}, err
	}

	val := iv.value
	if expr != "" {
		e, err := parser.ParseExpr("expression", expr)
		if err != nil {
			return val, errors.Wrap(err, "parse expression")
		}
		val = iv.value.Context().BuildExpr(e,
			cue.Scope(iv.value),
			cue.ImportPath(iv.instance.ID()),
			cue.InferBuiltins(true),
		)
		if val.Err() != nil {
			return val, errors.Wrap(val.Err(), "build expression")
		}
	}
	b, err := val.MarshalJSON()
	if err != nil {
		return val, errors.Wrap(err, "marshal json")
	}
	err = protojson.Unmarshal(b, into)
	if err != nil {
		return val, errors.Wrap(err, "proto unmarshal")
	}
	return val, nil
}

// Run runs all tests and returns a consolidated error.
func (t *Tester) Run() error {
	var errs []error
	function, err := fn.New(fn.Options{Debug: t.config.Debug})
	if err != nil {
		return errors.Wrap(err, "create function executor")
	}
	codeBytes, err := runDefCommand(t.config.Package)
	if err != nil {
		return errors.Wrap(err, "create package script")
	}
	for _, tag := range t.config.TestTags {
		err := t.runTest(function, codeBytes, tag)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "test %s", tag))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%d of %d tests had errors", len(errs), len(t.config.TestTags))
	}
	return nil
}

func canonicalYAML(in proto.Message) (string, error) {
	b, err := protojson.Marshal(in)
	if err != nil {
		return "", err
	}
	var ret any
	err = json.Unmarshal(b, &ret)
	if err != nil {
		return "", err
	}
	b, err = yaml.Marshal(ret)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (t *Tester) runTest(f *fn.Cue, codeBytes []byte, tag string) (finalErr error) {
	_, _ = fmt.Fprintf(TestOutput, "> run test %q\n", tag)
	defer func() {
		if finalErr != nil {
			_, _ = fmt.Fprintf(TestOutput, "FAIL %s: %s\n", tag, finalErr)
		} else {
			_, _ = fmt.Fprintf(TestOutput, "PASS %s\n", tag)
		}
	}()

	requestVar := "#request"
	if t.config.RequestVar != "" {
		requestVar = t.config.RequestVar
	}

	var responseVar string
	switch t.config.ResponseVar {
	case ".":
		responseVar = ""
	case "":
		responseVar = "response"
	default:
		responseVar = t.config.ResponseVar
	}

	var val cue.Value
	var expected fnv1.RunFunctionResponse
	var err error
	if t.config.LegacyDesiredOnlyResponse {
		expected.Desired = &fnv1.State{}
		val, err = evalPackage(t.config.TestPackage, tag, responseVar, expected.Desired)
	} else {
		val, err = evalPackage(t.config.TestPackage, tag, responseVar, &expected)
	}
	if err != nil {
		return errors.Wrap(err, "evaluate expected")
	}

	var req fnv1.RunFunctionRequest
	_, err = evalPackage(t.config.TestPackage, tag, requestVar, &req)
	if err != nil {
		return errors.Wrap(err, "evaluate request")
	}

	opts := fn.EvalOptions{
		RequestVar:          requestVar,
		ResponseVar:         responseVar,
		DesiredOnlyResponse: t.config.LegacyDesiredOnlyResponse,
		Debug:               fn.DebugOptions{Enabled: t.config.Debug},
	}
	actual, err := f.Eval(&req, string(codeBytes), opts)
	if err != nil {
		return errors.Wrap(err, "evaluate package with test request")
	}

	expectedBytes, err := protojson.MarshalOptions{Indent: "  "}.Marshal(&expected)
	if err != nil {
		return errors.Wrap(err, "proto json marshal")
	}
	actualBytes, err := protojson.MarshalOptions{Indent: "  "}.Marshal(actual)
	if err != nil {
		return errors.Wrap(err, "proto json marshal")
	}

	assertionMode := AssertionModeDiff

	attr := val.Attribute("assertionMode")
	if attr.Err() == nil {
		assertionMode, err = assertionModeFromString(attr.Contents())
		if err != nil {
			return err
		}
	}

	switch assertionMode {
	case AssertionModeUnification:
		assertionScript := fmt.Sprintf("expected: %s\n#Actual: %s\nunified: #Actual & expected\n", expectedBytes, actualBytes)

		runtime := cuecontext.New()
		assertVal := runtime.CompileString(assertionScript)
		if assertVal.Err() != nil {
			return errors.Wrap(assertVal.Err(), "compile cue code")
		}

		if _, err := assertVal.MarshalJSON(); err != nil {
			return errors.Wrap(err, "marshal cue output")
		}
		// script compiles and marshals, so actual and expected are unifiable.
	case AssertionModeDiff:
		expectedString, err := canonicalYAML(&expected)
		if err != nil {
			return errors.Wrap(err, "serialize expected")
		}
		actualString, err := canonicalYAML(actual)
		if err != nil {
			return errors.Wrap(err, "serialize actual")
		}
		if expectedString == actualString {
			return nil
		}

		err = printDiffs(expectedString, actualString)
		if err != nil {
			_, _ = fmt.Fprintln(TestOutput, "error in running diff:", err)
		}
		return fmt.Errorf("expected did not match actual")
	}
	return nil
}
