package wasmexec

import "runtime"

func Current() (content []byte, err error) {
	return Version(runtime.Version())
}

func Version(version string) (content []byte, err error) {
	return
}
