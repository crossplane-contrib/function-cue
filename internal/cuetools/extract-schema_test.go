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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var expectedSchema = `
package schemas

// generated cue schema

import "strings"

info: {
	title:       *"generated cue schema" | string
	description: ""
}
// allow creation of one of more S3 buckets
#XS3BucketV1alpha1: {
	// desired state
	spec!: {
		// bucket creation parameters
		parameters!: {
			// additional buckets to create with the suffixes provided
			additionalSuffixes?: [...strings.MaxRunes(
				4) & strings.MinRunes(
				1)]

			// bucket region
			region!: string

			// tags to associate with all buckets
			tags?: close({
				[string]: string
			})
			...
		}
		...
	}

	// observed status
	status?: {
		// additional endpoints in the same order as additional suffixes
		additionalEndpoints?: [...string]

		// the ARN of the IAM policy created for accessing the buckets
		iamPolicyARN?: string

		// the URL of the bucket endpoint
		primaryEndpoint?: string
		...
	}
	...
}
`

func TestExtractSchema(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("testdata", "xs3bucket.yaml"))
	require.NoError(t, err)
	ret, err := ExtractSchema(bytes.NewReader(b), "schemas")
	require.NoError(t, err)
	assert.Equal(t, strings.TrimSpace(expectedSchema), strings.TrimSpace(string(ret)))
}
