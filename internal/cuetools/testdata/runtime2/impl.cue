package runtime

resources: main: resource: {
	foo: _request.observed.composite.resource.foo
	bar: "baz"
}

