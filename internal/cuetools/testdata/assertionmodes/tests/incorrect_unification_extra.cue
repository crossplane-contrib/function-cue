
@if(incorrect_unification_extra)
package tests

#request: observed: composite: resource: {
	foo: "bar"
}

#mode: "unification"

response: {
    desired: resources: main: resource: {
		extra: "value"
	}
} @assertionMode(unification)
