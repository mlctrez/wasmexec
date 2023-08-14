# wasmexec

For applications that need to provide the correct wasm_exec.js based on a go runtime version.

## Rationale

The $GOROOT/misc/wasm/wasm_exec.js file is included in the go installation, but is only accessible from the
filesystem.

This module provides a means to source the correct wasm_exec.js content programmatically.

The golang source repository is scanned nightly and the current tags in the repository are mapped to the correct content
at `misc/wasm/wasm_exec.js`.

## Example

```go

http.HandleFunc("/wasm_exec.js", func (writer http.ResponseWriter, request *http.Request) {
content, err := wasmexec.Current()
if err != nil {
writer.WriteHeader(http.StatusInternalServerError)
return
}
writer.Header().Set("Content-Type", "application/javascript")
_, _ = writer.Write(content)
})

```

[![Go Report Card](https://goreportcard.com/badge/github.com/mlctrez/wasmexec)](https://goreportcard.com/report/github.com/mlctrez/wasmexec)

created by [tigwen](https://github.com/mlctrez/tigwen)
