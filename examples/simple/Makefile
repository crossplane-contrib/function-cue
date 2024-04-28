# Licensed to Elasticsearch B.V. under one or more contributor
# license agreements. See the NOTICE file distributed with
# this work for additional information regarding copyright
# ownership. Elasticsearch B.V. licenses this file to you under
# the Apache License, Version 2.0 (the "License"); you may
# not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied.  See the License for the
# specific language governing permissions and limitations
# under the License.

SHELL=bash

CUE_VERSION?=v0.7.0
DYFF_VERSION?=v1.7.0

FN_CUE_VERSION?=v0.2.0
FN_IMAGE:=xpkg.upbound.io/crossplane-contrib/function-cue

MAKE=make --no-print-directory

CUE=./.bin/cue
DYFF=./.bin/dyff
FN_CUE_TOOLS=./.bin/fn-cue-tools

XP_LOCAL_CONTEXT?=kind

export GOBIN=$(shell pwd)/.bin
export PATH:=$(GOBIN):$(PATH)
export DYFF_OPTS=--color on --truecolor off
export FUNCTION_CUE_DIFF:=dyff between --omit-header

COMPOSITIONS:=$(shell cd pkg/compositions/ && ls -d */ | sed 's|/||')

.DEFAULT_GOAL=pcdef

.PHONY: pcdef
pcdef:
	@echo -e run "\033[36m"make help"\033[0m" to see available targets >&2
	@$(MAKE) precommit

.PHONY: precommit
precommit: schemas scripts helm format test ## run pre-commit steps (default)

.PHONY: lib
lib: $(CUE) ## generate all libraries
	@echo generating libraries... >&2
	rm -rf cue.mod/gen
	@set -e;for lib in $$(cat .cuelibs); do echo $$lib >&2; cue get go $$lib; done

.PHONY: schemas
schemas: $(FN_CUE_TOOLS) ## generate schemas
	@echo generating schemas... >&2
	@mkdir -p ./zz_generated/schemas
	@$(FN_CUE_TOOLS) openapi --pkg schemas ./pkg/api | \
		sed 's/additionalProperties: {}/additionalProperties: true,\n "x-kubernetes-preserve-unknown-fields": true/g' | \
		cue fmt -s - >./zz_generated/schemas/generated-schemas.cue

.PHONY: scripts
scripts: $(FN_CUE_TOOLS) ## generate scripts
	@echo generating scripts... >&2
	@mkdir -p ./zz_generated/scripts
	@set -e;for c in $(COMPOSITIONS); do \
  		$(FN_CUE_TOOLS) package-script --pkg scripts --var $$c --out-file ./zz_generated/scripts/$$c.cue ./pkg/compositions/$$c; \
  	done

.PHONY: test
test: $(FN_CUE_TOOLS) $(DYFF) ## run cue tests
	@echo running tests... >&2
	@set -e; for c in $(COMPOSITIONS); do \
  		echo $$c; \
  		$(FN_CUE_TOOLS) cue-test --test-dir ./tests/compositions/$$c ./pkg/compositions/$$c; \
  	done

.PHONY: format
format: $(CUE) ## format cue files
	@echo formatting cue files... >&2
	@find . -name \*cue | grep -v -e cue.mod | xargs dirname | sort -u | xargs cue fmt

.PHONY: render
render: $(CUE)
	@cue eval --out=text -e 'yaml.MarshalStream($(resource))' -t image=$(FN_IMAGE):$(FN_CUE_VERSION) ./pkg >$(file)
	@echo --- >>$(file)

.PHONY: helm
helm: schemas scripts ## generate helm artifacts
	@echo generating helm artifacts ...>&2
	@mkdir -p helm/zz_generated
	@$(MAKE) render resource=functions file=helm/zz_generated/functions.yaml
	@$(MAKE) render resource=xrds file=helm/zz_generated/xrds.yaml
	@$(MAKE) render resource=compositions file=helm/zz_generated/compositions.yaml

.PHONY: k8s
k8s: helm ## renders YAML for XRDs, functions, and compositions
	helm template --include-crds ./helm

.PHONY: k8s-apply
k8s-apply: helm ## applies all helm artifacts to the ${XP_LOCAL_CONTEXT} context
	helm template --include-crds ./helm | kubectl apply --context $(XP_LOCAL_CONTEXT) -f -

.PHONY: user-s3
user-s3: $(CUE) ## renders namespaces and claims as YAML for the S3 bucket claims
	@cue eval --out=text -e 'yaml.MarshalStream(s3_resources)' ./user

.PHONY: user-s3-apply
user-s3-apply: ## applies namespaces and claims for the S3 bucket claims to the ${XP_LOCAL_CONTEXT} context
	@cue eval --out=text -e 'yaml.MarshalStream(s3_resources)' ./user | kubectl apply --context $(XP_LOCAL_CONTEXT) -f -

.PHONY: user-map
user-map: $(CUE) ## renders namespaces and claims as YAML for the replciated map claims
	@cue eval --out=text -e 'yaml.MarshalStream(map_resources)' ./user

.PHONY: user-map-apply
user-map-apply: ## applies namespaces and claims for the replicated config map claims to the ${XP_LOCAL_CONTEXT} context
	@cue eval --out=text -e 'yaml.MarshalStream(map_resources)' ./user | kubectl apply --context $(XP_LOCAL_CONTEXT) -f -


.PHONY: clean
clean: $(CUE) ## cleans all generated files
	rm -rf cue.mod/gen zz_generated helm/zz_generated .bin/
	@$(MAKE) lib

$(CUE):
	go install cuelang.org/go/cmd/cue@$(CUE_VERSION)

$(FN_CUE_TOOLS):
	go install github.com/crossplane-contrib/function-cue/cmd/fn-cue-tools

$(DYFF):
	go install github.com/homeport/dyff/cmd/dyff@$(DYFF_VERSION)

.PHONY: help
help: ## displays this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
