@if(correct)
package tests

#request: observed: composite: resource: {
	foo: "bar"
}

response: desired: resources: main: resource: {
		foo: "bar"
		bar: "baz"
}
