package wasmexec

import "net/http"

// WriteLauncher writes Current wasm js and minimal WebAssembly instantiation code.
func WriteLauncher(writer http.ResponseWriter) {
	var data []byte
	var err error
	if data, err = Current(); err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	data = append(data, []byte(appJs)...)
	writer.Header().Set("Content-Type", "application/javascript")
	_, _ = writer.Write(data)
}

var appJs = `
//
// web assembly launcher
//
(() => {
  const go = new Go();
  WebAssembly.instantiateStreaming(fetch("app.wasm"), go.importObject)
    .then((result) => {
      go.run(result.instance)
        .then(() => console.log("go.run exited"))
        .catch(err => console.log("error ", err));
    }).catch(err => console.log("error ", err))
})();
`
