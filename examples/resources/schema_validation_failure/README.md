# [xrender](https://github.com/crossplane-contrib/xrender)

#### Run
```bash
$ xrender xr.yaml composition.yaml functions.yaml
```

#### Produces Schema Validation Failure
```
xrender: error: main.CLI.Run(): cannot render composite resource: pipeline step "schema" returned a fatal result: failed compiling cue template: failed creating cue compiler: failed to validate: spec.parameters: field not allowed
```
