# wasmexec

For applications that need to provide the correct wasm_exec.js based on a go runtime version.

## Rationale

The $GOROOT/misc/wasm/wasm_exec.js file is included in the go installation, but is only accessible from the
filesystem.

This module provides a means to source the correct wasm_exec.js content programmatically.

## Example

```go

content, err = wejs.Current()

```

[![Go Report Card](https://goreportcard.com/badge/github.com/mlctrez/wasmexec)](https://goreportcard.com/report/github.com/mlctrez/wasmexec)

created by [tigwen](https://github.com/mlctrez/tigwen)
