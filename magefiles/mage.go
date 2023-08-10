package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"fmt"
	"github.com/magefile/mage/sh"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var Default = Build

func Build() (err error) {

	var projectDir string
	projectDir, err = os.Getwd()
	if err != nil {
		return
	}

	tempDir := "/tmp/gitTemp"
	repoDir := filepath.Join(tempDir, "go")
	wasmExecPath := "misc/wasm/wasm_exec.js"
	wasmExecFile := filepath.Join(repoDir, wasmExecPath)

	if err = os.MkdirAll(tempDir, 0755); err != nil {
		return
	}
	if err = os.Chdir(tempDir); err != nil {
		return
	}
	if _, err = os.Stat(filepath.Join(repoDir, ".git")); os.IsNotExist(err) {
		err = sh.Run("git", "clone", "https://github.com/golang/go")
	}
	if err != nil {
		return
	}
	if err = os.Chdir(repoDir); err != nil {
		return
	}
	if err = sh.Run("git", "fetch"); err != nil {
		return
	}
	var output string
	if output, err = sh.Output("git", "tag"); err != nil {
		return
	}
	var goTags []string
	scanner := bufio.NewScanner(bytes.NewBufferString(output))
	for scanner.Scan() {
		tagName := scanner.Text()
		if strings.HasPrefix(tagName, "go") {
			if tagName > "go1.16" {
				goTags = append(goTags, tagName)
			}
		}
	}

	shaMapping := map[string][]string{}
	contentMapping := map[string][]byte{}

	for _, tag := range goTags {

		checkout := exec.Command("git", "checkout", tag, wasmExecPath)
		var out []byte
		out, err = checkout.CombinedOutput()
		if err != nil {
			if strings.Contains(string(out), "did not match any file(s) known to git") {
				continue
			}
			fmt.Println(tag, string(out))
			return err
		}

		_, err = os.Stat(wasmExecFile)
		if os.IsNotExist(err) {
			continue
		}

		var wasmExecBytes []byte
		wasmExecBytes, err = os.ReadFile(wasmExecFile)

		h := sha256.New()
		h.Write(wasmExecBytes)
		bs := h.Sum(nil)
		shaSum := fmt.Sprintf("%x", bs)

		if m, ok := shaMapping[shaSum]; ok {
			m = append(m, tag)
			shaMapping[shaSum] = m
		} else {
			shaMapping[shaSum] = []string{tag}
		}
		contentMapping[shaSum] = wasmExecBytes

	}

	if err = os.Chdir(projectDir); err != nil {
		return
	}

	weJs := bytes.NewBufferString("")

	_, _ = weJs.WriteString("package wasmexec\n")
	_, _ = weJs.WriteString("import \"fmt\"\n")
	_, _ = weJs.WriteString("import \"runtime\"\n")

	_, _ = weJs.WriteString("func Current() (content []byte, err error) {\n")
	_, _ = weJs.WriteString("return Version(runtime.Version())\n")
	_, _ = weJs.WriteString("}\n\n")

	_, _ = weJs.WriteString("func Version(version string) (content []byte, err error) {\n")
	_, _ = weJs.WriteString("\n")
	_, _ = weJs.WriteString("switch version{\n")

	for k, tags := range shaMapping {
		_, _ = weJs.WriteString("\n")
		caseStatement := fmt.Sprintf("case \"%s\" :\n", strings.Join(tags, `", "`))
		_, _ = weJs.WriteString(caseStatement)
		_, _ = weJs.WriteString(fmt.Sprintf("return []byte(%q), nil\n", contentMapping[k]))
	}

	_, _ = weJs.WriteString("\n")
	_, _ = weJs.WriteString("default :\n")
	_, _ = weJs.WriteString("return nil, fmt.Errorf(\"unknown version %s\", version)\n")
	_, _ = weJs.WriteString("}\n")
	_, _ = weJs.WriteString("}\n")

	var weJsFormatted []byte
	weJsFormatted, err = format.Source(weJs.Bytes())

	err = os.WriteFile("versions.go", weJsFormatted, 0644)

	return
}
