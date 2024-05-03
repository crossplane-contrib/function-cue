@if(incorrect)
package tests

#request: observed: composite: resource: {
	foo: "foo2"
}

response: desired: resources: main: resource: {
		foo: "bar"
		bar: "baz"
}
