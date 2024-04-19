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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	godiff "github.com/pmezard/go-difflib/difflib"
)

const (
	ExternalDiffEnvVar = "FUNCTION_CUE_DIFF"
)

func printNativeDiffs(expectedString, actualString string) error {
	ud := godiff.UnifiedDiff{
		A:        godiff.SplitLines(expectedString),
		B:        godiff.SplitLines(actualString),
		FromFile: "expected",
		ToFile:   "actual",
		Context:  3,
	}
	s, err := godiff.GetUnifiedDiffString(ud)
	if err != nil {
		return errors.Wrap(err, "diff expected against actual")
	}
	_, _ = fmt.Fprintf(TestOutput, "diffs found:\n%s\n", strings.TrimSpace(s))
	return nil
}

func printDiffs(expectedString, actualString string) error {
	externalDiff := os.Getenv(ExternalDiffEnvVar)
	if externalDiff == "" {
		return printNativeDiffs(expectedString, actualString)
	}
	// use the logic of external kubectl diffs to run the command
	args := strings.Split(externalDiff, " ")
	cmd := args[0]
	args = args[1:]
	var realArgs []string
	isValidChar := regexp.MustCompile(`^[a-zA-Z0-9-=]+$`).MatchString
	for _, arg := range args {
		if isValidChar(arg) {
			realArgs = append(realArgs, arg)
		}
	}

	dir, err := os.MkdirTemp("", "diff*")
	if err != nil {
		return err
	}
	defer func() {
		_ = os.RemoveAll(dir)
	}()
	eFile := filepath.Join(dir, "expected.yaml")
	err = os.WriteFile(eFile, []byte(expectedString), 0o644)
	if err != nil {
		return err
	}
	aFile := filepath.Join(dir, "actual.yaml")
	err = os.WriteFile(aFile, []byte(actualString), 0o644)
	if err != nil {
		return err
	}
	realArgs = append(realArgs, eFile, aFile)
	c := exec.Command(cmd, realArgs...)
	c.Stdout = TestOutput
	c.Stderr = TestOutput
	return c.Run()
}
