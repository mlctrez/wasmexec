package wasmexec

import (
	"runtime"
	"testing"
)

func TestVersion(t *testing.T) {
	var contents []byte
	var err error
	for version, sha := range tagToShaMap {
		if contents, err = readContents(version); err != nil {
			t.Fatal(err)
		}
		if sha != shaString(contents) {
			t.Fatal("sha mismatch", sha, shaString(contents))
		}
	}
}

func TestCurrent(t *testing.T) {
	content, err := Current()
	if err != nil {
		t.Fatal(err)
	}
	currentSha := shaString(content)
	currentFromRuntime, err := readContents(runtime.Version())
	if err != nil {
		t.Fatal(err)
	}
	if currentSha != shaString(currentFromRuntime) {
		t.Fatal("mismatch between current and runtime current")
	}
}
