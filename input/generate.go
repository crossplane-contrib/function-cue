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

//go:build generate

// Remove existing and generate new input manifests
//go:generate rm -rf ../package/input/
//go:generate go run -tags generate sigs.k8s.io/controller-tools/cmd/controller-gen paths=./v1beta1 object crd:crdVersions=v1 output:artifacts:config=../package/input

package input

import (
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
)
