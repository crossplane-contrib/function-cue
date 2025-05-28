@if(unification)
package tests

#request: observed: composite: resource: {
	foo: "bar"
}

response: {
    desired: resources: main: resource: {
		foo: "bar"
	}
} @assertionMode(unification)
