# Licensed to Elasticsearch B.V. under one or more contributor
# license agreements. See the NOTICE file distributed with
# this work for additional information regarding copyright
# ownership. Elasticsearch B.V. licenses this file to you under
# the Apache License, Version 2.0 (the "License"); you may
# not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied.  See the License for the
# specific language governing permissions and limitations
# under the License.

image?=xpkg.upbound.io/crossplane-contrib/function-cue

build_date:=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
commit:=$(shell git rev-parse --short HEAD 2> /dev/null)
bdate:=$(shell date -u +%Y%m%d%H%M%S)
bver:=$(shell git rev-parse --short=12 HEAD)
version:=$(shell git describe --tags --exact-match --match='v*' 2> /dev/null || echo "$(bdate)-$(bver)")
ldflags?=-X 'main.BuildDate=$(build_date)' -X 'main.Commit=$(commit)' -X 'main.Version=$(version)'

.PHONY: local
local: build test lint

.bin/gofumpt:
	mkdir -p ./.bin
	GOBIN="$$(pwd)/.bin" go install mvdan.cc/gofumpt@v0.5.0

.bin/golangci-lint:
	mkdir -p ./.bin
	GOBIN="$$(pwd)/.bin" go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.59.1

.PHONY: build
build:
	CGO_ENABLED=0 go generate ./...
	CGO_ENABLED=0 go install -ldflags="$(ldflags)" ./...

.PHONY: test
test:
	go test -race ./...

.PHONY: lint
lint: .bin/golangci-lint
	./.bin/golangci-lint run

.PHONY: docker-build
docker-build:
	docker build --tag $(image):$(version) --build-arg "ldflags=$(ldflags)" .

.PHONY: docker
docker:
	make -C . docker-build

.PHONY: docker-push
docker-push: docker
	docker push $(image):$(version)

.PHONY: fmt
fmt: .bin/gofumpt
	./.bin/gofumpt -w .

.PHONY: ci
ci: local

.PHONY: ci-print-ldflags
ci-print-ldflags:
	@echo $(ldflags)

.PHONY: ci-print-version
ci-print-version:
	@echo $(version)

.PHONY: ci-check-dirty
ci-check-dirty:
	git status || true
	git diff --quiet || (echo 'dirty files found' && exit 1)
