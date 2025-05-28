
@if(incorrect_unification)
package tests

#request: observed: composite: resource: {
	foo: "bar"
}

response: {
    desired: resources: main: resource: {
		foo: "baz"
	}
} @assertionMode(unification)
