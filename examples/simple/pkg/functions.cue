package pkg

import (
	xp "github.com/crossplane/crossplane/apis/pkg/v1beta1"
)

_functions: cuefn: xp.#Function & {
	apiVersion: "pkg.crossplane.io/v1beta1"
	kind:       "Function"
	metadata: {
		name: "fn-cue-examples-simple"
	}
	spec: {
		package:           string | *"docker.io/gotwarlost/function-cue:test" @tag(image)
		packagePullPolicy: "Always"
	}
}

_functions: readyFn: xp.#Function & {
	apiVersion: "pkg.crossplane.io/v1beta1"
	kind:       "Function"
	metadata: name: "fn-auto-ready"
	spec: {
		package: "xpkg.upbound.io/crossplane-contrib/function-auto-ready:v0.2.1"
	}
}
