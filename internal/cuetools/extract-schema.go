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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/encoding/openapi"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
)

type expectedXRDVersion struct {
	Name   string `json:"name"`
	Schema struct {
		OpenAPIV3Schema any `json:"openAPIV3Schema"`
	} `json:"schema"`
}

type expectedXRD struct {
	Spec struct {
		Names struct {
			Kind string `json:"kind"`
		} `json:"names"`
		Versions []*expectedXRDVersion `json:"versions"`
	} `json:"spec"`
}

type openapiSchema struct {
	OpenAPI string `json:"openapi"`
	Info    struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	} `json:"info"`
	Components struct {
		Schemas map[string]any `json:"schemas"`
	} `json:"components"`
}

// ExtractSchema extracts an openAPI schema from a CRD/ XRD-like object and
// returns the equivalent cue types.
func ExtractSchema(reader io.Reader, pkg string) ([]byte, error) {
	b, err := io.ReadAll(reader)
	if err != nil {
		return nil, errors.Wrap(err, "read bytes")
	}
	var xrd expectedXRD
	err = yaml.Unmarshal(b, &xrd)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal crd/xrd")
	}
	var pkgDecl string
	if pkg != "" {
		pkgDecl = fmt.Sprintf("package %s\n\n", pkg)
	}
	out := bytes.NewBufferString(pkgDecl)
	for _, version := range xrd.Spec.Versions {
		var cueSchema openapiSchema
		cueSchema.OpenAPI = "3.0.0"
		cueSchema.Info.Title = "generated cue schema"
		v := version.Name
		if len(v) > 0 {
			v = strings.ToUpper(v[:1]) + v[1:]
		}
		cueSchema.Components.Schemas = map[string]any{
			fmt.Sprintf("%s%s", xrd.Spec.Names.Kind, v): version.Schema.OpenAPIV3Schema,
		}
		jsonBytes, err := json.MarshalIndent(cueSchema, "", "  ")
		if err != nil {
			return nil, errors.Wrap(err, "marshal schema")
		}
		runtime := cuecontext.New()
		val := runtime.CompileBytes(jsonBytes)
		if val.Err() != nil {
			return nil, errors.Wrap(val.Err(), "compile generated schema object")
		}
		astFile, err := openapi.Extract(val, &openapi.Config{SelfContained: true})
		if err != nil {
			return nil, errors.Wrap(err, "extract openAPI schema")
		}
		b, err = format.Node(astFile, format.Simplify())
		if err != nil {
			return nil, errors.Wrap(val.Err(), "format source")
		}
		_, err = out.Write(b)
		if err != nil {
			return nil, errors.Wrap(err, "write bytes")
		}
		_, _ = out.Write([]byte("\n"))
	}
	return out.Bytes(), nil
}
