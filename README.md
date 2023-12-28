# function-cue

A [cuelang](https://cuelang.org/) integration for Crossplane's [Composition Functions](https://docs.crossplane.io/latest/concepts/composition-functions/).

This function is based on [/play](https://cuelang.org/play) [codebase](https://github.com/cue-lang/cuelang.org/blob/master/play/main.go) and [cmd](https://github.com/cue-lang/cue/blob/master/cmd/cue/cmd/export.go)

## Purpose

This function is intended to give a crossplane composition creator the ability to compile and run cue templates within a composition function.
The user can target the compilation output to various objects determined by the target passed into the `CUEInput.Export.Target`

#### Targeting Patches

See [Targeting Objects](docs/TARGETING_OBJECTS.md) for info on utilizing the types of targets

```
Resources (default)
PatchDesired
PatchResources
XR
```

## Expected Function Input

See the kubebuilder generated [CRD](package/input/cue.fn.crossplane.io_cueinputs.yaml) or the [go definition](input/v1beta1/input.go)

#### Example Compositions

See [examples folder](examples)

## Expected Output

The compilation output of the `CUEInput.Export.Value` **must** be in `YAML` or `JSON` documents, or it will fail parsing.

- Each document produced should be a valid crossplane resource `xr` or `mr`
- Each document must have an `apiVersion`, `kind`

Ex `cueCompile()` output schema

```cue
#outputSchema: {
	// Name of your resource
	name: string
	// Optional connection details
	// see docs/CONNECTION_DETAILS.md for more details
	connectionDetails?: [...#connectionDetail]
	// Optional readiness checks
	// see docs/READINESS_CHECKS.md for more details
	readinessChecks?: [...#readinessCheck]
	// Resource spec
	resource: {
		// Example
		// apiVersion: "nobu.dev/v1"
		// kind: "Example"
		//
		// etc...
	}
}
```

#### json
`single`
```json
{
  "name": "my-cluster",
  "resource": {
    "apiVersion": "nobu.dev/v1",
    "kind": "XCluster"
  }
}
```

`multiple` - single json document per line, `-e json.MarshalStream(field)` will produce this format
```json
{"name": "my-cluster", "resource": {"apiVersion": "nobu.dev/v1", "kind": "XCluster"}}
{"name": "my-network", "resource": {"apiVersion": "nobu.dev/v1", "kind": "XNetwork"}}
```

#### yaml
`single`
```yaml
name: "my-cluster"
resource:
  apiVersion: "nobu.dev/v1"
  kind: "XCluster"
```

`multiple` documents, separated by `---`, `-e yaml.MarshalStream(field)` will produce this format
```yaml
name: "my-cluster"
resource:
  apiVersion: "nobu.dev/v1"
  kind: "XCluster"
---
name: "my-network"
resource:
  apiVersion: "nobu.dev/v1"
  kind: "XNetwork"
...
```

## CUE Version supported

`v0.7.0`

## Command Support

### `export`

Runs `cue export` based off of a cue template provided by the `CUEInput.Export.Value` field

See the currently supported [options](docs/EXPORT_OPTIONS.md)

## Installing

There are two different versions that are built, with the `cue.mod` and without

Without cue.mod, if you do not want schema validation, this is not needed
```yaml
apiVersion: pkg.crossplane.io/v1beta1
kind: Function
metadata:
  name: function-cue
spec:
  package: mitsuwa/function-cue:v0.1.2
```

With cue.mod, the cue.mod is its own layer so it should get cached, currently it is `~164M`
```yaml
apiVersion: pkg.crossplane.io/v1beta1
kind: Function
metadata:
  name: function-cue
spec:
  package: mitsuwa/function-cuemod:v0.1.2
```

## Importable Packages

See [libraries](docs/LIBRARIES.md#cue.mod)

## Debugging

Logs are emitted to the Function's pod logs. Look for the Function pod in `crossplane-system`.

#### Levels

```
Info   # default
Debug  # run with --debug flag
```
