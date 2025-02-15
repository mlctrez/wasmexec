package main

import (
	"fmt"
	"github.com/mlctrez/wasmexec/gitutil"
	"github.com/mlctrez/wasmexec/gitver"
	"github.com/mlctrez/wasmexec/shautil"
	"github.com/mlctrez/wasmexec/sourcefile"
	"github.com/pkg/errors"
	"os"
	"os/exec"
)

var Default = Build

func Build() (err error) {

	repository := "https://github.com/golang/go"
	tempDir := "/tmp/golang"
	wasmExecPaths := []string{"misc/wasm/wasm_exec.js", "lib/wasm/wasm_exec.js"}

	gv := gitver.New(repository, tempDir, wasmExecPaths, "go")
	if err = gv.Run(); err != nil {
		return
	}

	sf := sourcefile.New()

	if err = gv.Versions(sf); err != nil {
		return err
	}

	var content []byte
	if content, err = sf.Format(); err != nil {
		return err
	}

	newSum := shautil.ShaString(content)

	var oldSum string
	if _, oldSum, err = shautil.ReadWithSha("versions.go"); err != nil {
		return
	}

	if newSum == oldSum {
		fmt.Println("no changes to file, exiting")
		return
	}

	if err = os.WriteFile("versions.go", content, 0644); err != nil {
		return
	}

	var testOutput []byte
	testOutput, err = exec.Command("go", "test").CombinedOutput()
	if err != nil {
		return errors.WithMessage(err, string(testOutput))
	}

	var gu *gitutil.GitUtil
	if gu, err = gitutil.Open("."); err != nil {
		return
	}

	gu.Signature("mlctrez", "mlctrez@gmail.com")

	if err = gu.Add("versions.go", "ci update"); err != nil {
		return
	}

	return gu.PushNewVersion()
}
