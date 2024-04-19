package compositions

import (
	xp "github.com/crossplane/crossplane/apis/apiextensions/v1"
	schemas "cue-functions.io/examples/simple/zz_generated/schemas"
	scripts "cue-functions.io/examples/simple/zz_generated/scripts"
)

let version = "v1alpha1"
let pluralName = "xs3buckets"
let xrdKind = "XS3Bucket"
let groupName = "simple.cuefn.example.com"

_xrds: s3Bucket: xp.#CompositeResourceDefinition & {
	apiVersion: "apiextensions.crossplane.io/v1"
	kind:       "CompositeResourceDefinition"
	metadata: {
		name: "\(pluralName).\(groupName)"
	}

	spec: {
		group: groupName
		names: {
			kind:   xrdKind
			plural: pluralName
		}
		claimNames: {
			kind:   "S3Bucket"
			plural: "s3buckets"
		}
		versions: [{
			name:          "v1alpha1"
			served:        true
			referenceable: true
			additionalPrinterColumns: [{
				jsonPath: ".status.APIEndpoint"
				name:     "Endpoint"
				type:     "string"
			}, {
				jsonPath: ".spec.resourceRef.name"
				name:     "external-name"
				type:     "string"
			}]
			schema: openAPIV3Schema: schemas.components.schemas.S3BucketV1alpha1
		}]
	}
}

_compositions: s3Bucket: xp.#Composition & {
	let fullName = "\(pluralName).\(groupName)"
	apiVersion: "apiextensions.crossplane.io/v1"
	kind:       "Composition"
	metadata: {
		name: fullName
		labels: {
			"crossplane.io/xrd": fullName
		}
	}
	spec: {
		compositeTypeRef: {
			apiVersion: "\(groupName)/\(version)"
			kind:       xrdKind
		}
		resources: []
		mode: "Pipeline"
		pipeline: [
			{
				step: "run cue composition"
				functionRef: name: "fn-cue-examples-simple"
				input: {
					apiVersion: "function-cue/v1"  // value does not matter
					kind:       "CueFunctionInput" // ditto
					source:     "Inline"
					script:     scripts.s3bucket
					debugNew:   true
				}
			},
			{
				step: "run auto ready"
				functionRef: name: "fn-auto-ready"
			},
		]
	}
}
