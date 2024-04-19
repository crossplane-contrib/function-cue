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

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/crossplane-contrib/function-cue/internal/cuetools"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func checkNoArgs(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("no arguments are expected for this command: found %v", args)
	}
	cmd.SilenceUsage = true
	return nil
}

func checkOneArg(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("exactly one argument is expected for this command, got %d,  %v", len(args), args)
	}
	cmd.SilenceUsage = true
	return nil
}

func writeFile(file string, content []byte) error {
	dir := filepath.Dir(file)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(file, content, 0o644)
}

func openapiCommand() *cobra.Command {
	var pkg, outFile string
	c := &cobra.Command{
		Use:   "openapi ./path/to/package/dir",
		Short: "generate self-contained openapi schemas for cue types from a cue package",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkOneArg(cmd, args); err != nil {
				return err
			}
			out, err := cuetools.GenerateOpenAPISchema(args[0], pkg)
			if err != nil {
				return errors.Wrap(err, "generate schemas")
			}
			if outFile == "" || outFile == "-" {
				fmt.Println(string(out))
				return nil
			}
			return writeFile(outFile, out)
		},
	}
	f := c.Flags()
	f.StringVar(&pkg, "pkg", "schemas", "package name of generated cue file")
	f.StringVar(&outFile, "out-file", "", "output file name, default is stdout")
	return c
}

func packageScriptCommand() *cobra.Command {
	var pkg, outFile, out, varName string
	c := &cobra.Command{
		Use:   "package-script ./path/to/package/dir",
		Short: "generate a self-contained script as text",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkOneArg(cmd, args); err != nil {
				return err
			}
			out, err := cuetools.PackageScript(args[0], cuetools.PackageScriptOpts{
				VarName:       varName,
				OutputPackage: pkg,
				Format:        cuetools.OutputFormat(out),
			})
			if err != nil {
				return errors.Wrap(err, "generate schemas")
			}
			if outFile == "" || outFile == "-" {
				fmt.Println(string(out))
				return nil
			}
			return writeFile(outFile, out)
		},
	}
	f := c.Flags()
	f.StringVar(&pkg, "pkg", "", "package name of generated cue file")
	f.StringVar(&varName, "var", "_script", "the variable name to use for the script, cue format only")
	f.StringVar(&outFile, "out-file", "", "output file name, default is stdout")
	f.StringVarP(&out, "output", "o", string(cuetools.FormatCue), "output format, one of cue or raw")
	return c
}

func extractSchemaCommand() *cobra.Command {
	var pkg, file, outFile string
	c := &cobra.Command{
		Use:   "extract-schema",
		Short: "extract a cue schema from the openAPI spec of a CRD/XRD object",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkNoArgs(cmd, args); err != nil {
				return err
			}
			var reader io.Reader
			if file == "" || file == "-" {
				reader = os.Stdin
			} else {
				f, err := os.Open(file)
				if err != nil {
					return err
				}
				defer func() { _ = f.Close() }()
				reader = f
			}
			out, err := cuetools.ExtractSchema(reader, pkg)
			if err != nil {
				return errors.Wrap(err, "generate schemas")
			}
			if outFile == "" || outFile == "-" {
				fmt.Println(string(out))
				return nil
			}
			return writeFile(outFile, out)
		},
	}
	f := c.Flags()
	f.StringVar(&pkg, "pkg", "", "package name of generated cue file")
	f.StringVar(&file, "file", "-", "input JSON or YAML file containing a single CRD/ XRD definition, defaults to stdin")
	f.StringVar(&outFile, "out-file", "", "output file name, defaults to stdout")
	return c
}

func cueTestCommand() *cobra.Command {
	var p cuetools.TestConfig
	c := &cobra.Command{
		Use:   "cue-test ./path/to/package/dir",
		Short: "run unit tests for your composition implementation",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkOneArg(cmd, args); err != nil {
				return err
			}
			p.Package = args[0]
			tester, err := cuetools.NewTester(p)
			if err != nil {
				return err
			}
			return tester.Run()
		},
	}
	f := c.Flags()
	f.StringSliceVar(&p.TestTags, "tag", nil, "list of test tags to enable, one per test")
	f.StringVar(&p.TestPackage, "test-dir", "", "relative path to test package, defaults to a tests subdirectory under the package")
	f.StringVar(&p.RequestVar, "request", "#request", "request variable to inject")
	f.StringVar(&p.ResponseVar, "response", "response", "response variable to extract")
	f.BoolVar(&p.LegacyDesiredOnlyResponse, "legacy-response", false, "enable legacy response")
	f.BoolVar(&p.Debug, "debug", false, "enable eval debugging")
	return c
}
