# function-cue

A crossplane function that runs cue scripts for composing resources. 
Contributed by [Elastic](https://github.com/elastic) from their [initial implementation](https://github.com/elastic/crossplane-function-cue).

#### Build status

![CI](https://github.com/crossplane-contrib/function-cue/actions/workflows/ci.yaml/badge.svg?branch=main)
[![Go Report Card](https://goreportcard.com/badge/github.com/crossplane-contrib/function-cue)](https://goreportcard.com/report/github.com/crossplane-contrib/function-cue)
[![Go Coverage](https://github.com/crossplane-contrib/function-cue/wiki/coverage.svg)](https://raw.githack.com/wiki/elastic/function-cue/coverage.html)

## Building

```shell
$ make # generate input, compile, test, lint
$ make docker # build docker image
```

## Function interface

You define the function as follows:
```yaml
apiVersion: pkg.crossplane.io/v1beta1
kind: Function
metadata:
  name: fn-cue
spec:
  package: crossplane-contrib/function-cue:version
```

and reference it in a composition as follows:

```yaml
  pipeline:
    - step: run cue composition
      functionRef:
        name: fn-cue
      input:
        apiVersion: fn-cue/v1    # can be anything
        kind: CueFunctionParams  # can be anything
        source: Inline         # only Inline is supported for now
        script: |              # text of cue program
          text of cue program
        # show inputs and outputs for the composition in the pod log in pretty format
        debug: true  
```

The full spec of the input object can be [found here](input/v1beta1/input.go)

## The Go code

The [main program](main.go) at the root of the repo provides the function implementation.

The program [fn-cue-tools](cmd/fn-cue-tools) provides additional support tooling with the following subcommands.

* `openapi` - utility that converts a cue type into an openAPI schema that has self-contained types.
* `extract-schema` - convert an existing openAPI schema found in a CRD/ XRD YAML to cue.
* `package-script` - utility that takes a cue package and turns it into a self-contained cue script of the form:
* `cue-test` - utility to unit test your cue implementation using inputs from various stages of the composition lifecycle.

## The cue script

The cue script is a single self-contained program(*) that you provide which is compiled after it is appended with 
additional cue code that looks like the following:

```
  "#request": <input-object>
```

The &lt;input-object&gt; is the same as the [RunFunctionRequest](https://github.com/crossplane/crossplane/blob/bf5c51e6dfdde4c45a0d50c31c23147f5050e9dd/apis/apiextensions/fn/proto/v1beta1/run_function.proto#L33) 
message in JSON form, except it only contains the `observed`, `desired`, and `context` attributes. 
It does **not** have the `meta` or the `input` attributes.

The cue script is expected to return a response that is the JSON equivalent of the [RunFunctionResponse](https://github.com/crossplane/crossplane/blob/bf5c51e6dfdde4c45a0d50c31c23147f5050e9dd/apis/apiextensions/fn/proto/v1beta1/run_function.proto#L66)
message containing the desired state and optionally context variables to be set for the pipeline. 
The function runner will selectively update its internal desired state with the
returned resources. If a composite is returned, it will also be set in the response. You will only typically include the
`status` of the composite resource.

(*) Note that it is not necessary for the cue source code to be in a single file. It can span multiple files in a single
package and depend on other packages. You use the `package-script` sub-command of `fn-cue-tools` to create the
self-contained script. This, in turn, uses `cue def --inline-imports` under the covers.

The names of the request and response objects are configurable in the function input.

See the [example implementation](examples/simple/pkg/compositions/s3bucket) to get a sense of 
how the composition works. A detailed walkthrough can be found in the [README](examples/simple/) for the example.

## Debug output for specific XRs

The function can produce debug output in terms of showing requests and responses in the pod logs, which is also
useful for setting up unit tests. You can enable debugging per-XR by annotating it as follows:

```
cue.fn.crossplane.io/debug=true
```

## License

The code is distributed under the Apache 2 license. See the [LICENSE](LICENSE) file for details.
