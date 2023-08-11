package wasmexec

// generated from https://github.com/mlctrez/wasmexec

import (
	"fmt"
	"runtime"
)

func Current() (content []byte, err error) {
	return Version(runtime.Version())
}

func Version(version string) (content []byte, err error) {

	if contentFunc, ok := versionMap[version]; ok {
		content = []byte(contentFunc())
	} else {
		err = fmt.Errorf("unsupported version %q", version)
	}

	return
}

var versionMap = map[string]func() string{}
