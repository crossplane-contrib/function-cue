package pkg

import (
	xp "github.com/crossplane/crossplane/apis/pkg/v1beta1"
)

_functions: cuefn: xp.#Function & {
	apiVersion: "pkg.crossplane.io/v1"
	kind:       "Function"
	metadata: {
		name: "fn-cue-examples-simple"
	}
	spec: {
		package:           string | *"xpkg.upbound.io/crossplane-contrib/function-cue:@v0.2.0" @tag(image)
		packagePullPolicy: "Always"
	}
}

_functions: readyFn: xp.#Function & {
	apiVersion: "pkg.crossplane.io/v1"
	kind:       "Function"
	metadata: name: "fn-auto-ready"
	spec: {
		package: "xpkg.upbound.io/crossplane-contrib/function-auto-ready:v0.2.1"
	}
}
