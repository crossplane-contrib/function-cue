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
	"bytes"
	"fmt"
	"runtime"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// Build information. Populated at build-time.
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

// info contains detailed information about the binary.
type buildInfo struct {
	version, commit, buildDate string
}

func (v buildInfo) String() string {
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 2, 2, 3, ' ', 0)
	write := func(prompt, value string) {
		_, _ = fmt.Fprintln(w, prompt+"\t", value)
	}
	write("Version", v.version)
	write("Go Version", runtime.Version())
	write("Commit", strings.ReplaceAll(v.commit, "_", " "))
	write("Build Date", v.buildDate)
	write("OS/Arch", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH))
	_ = w.Flush()
	return buf.String()
}

func versionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "print program version",
		Run: func(cmd *cobra.Command, args []string) {
			info := buildInfo{Version, Commit, BuildDate}
			fmt.Println(info)
		},
	}
}
