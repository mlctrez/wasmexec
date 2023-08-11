package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/mlctrez/wasmexec/gitutil"
	"github.com/rogpeppe/go-internal/semver"
	"go/format"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

var Default = Build

func Build() (err error) {
	tag, err := incrementMinor()
	if err != nil {
		return
	}
	fmt.Println(tag)
	return
}

func incrementMinor() (tag string, err error) {
	var repo *git.Repository
	if repo, err = git.PlainOpen("."); err != nil {
		return
	}

	var tags storer.ReferenceIter
	tags, err = repo.Tags()
	var sortedTags []string
	err = tags.ForEach(func(reference *plumbing.Reference) error {
		if semver.IsValid(reference.Name().Short()) {
			sortedTags = append(sortedTags, reference.Name().Short())
		}
		return nil
	})
	if err != nil {
		return
	}

	sort.SliceStable(sortedTags, func(i, j int) bool {
		return semver.Compare(sortedTags[i], sortedTags[j]) > 0
	})

	latest := sortedTags[0]

	split := strings.Split(latest, ".")

	atoi, err := strconv.Atoi(split[2])
	if err != nil {
		return
	}

	split[2] = fmt.Sprintf("%s", fmt.Sprintf("%d", atoi+1))

	tag = fmt.Sprintf("v%s.%s.%s", split[0], split[1], split[2])

	return
}

func BuildOld() (err error) {

	gu := gitutil.New("https://github.com/golang/go", "/tmp/golang")

	if err = gu.CloneOrOpen(); err != nil {
		return
	}

	wasmExecPath := "misc/wasm/wasm_exec.js"

	var refs []*plumbing.Reference
	if refs, err = gu.Tags("go"); err != nil {
		return
	}

	var shaMapping = make(map[string]string)
	var contentMapping = make(map[string][]byte)

	for _, ref := range refs {

		if !strings.HasPrefix(ref.Name().Short(), "go1.2") {
			continue
		}

		var content []byte
		content, err = gu.Contents(wasmExecPath, ref)
		if os.IsNotExist(err) {
			continue
		}
		fmt.Printf("getting content for ref %s\n", ref.Name().Short())
		sum := shaContents(content)
		contentMapping[sum] = content
		shaMapping[ref.Name().Short()] = sum
	}

	var wasmExecContent []byte
	if wasmExecContent, err = buildWasmExec(shaMapping, contentMapping); err != nil {
		return
	}

	newSum := shaContents(wasmExecContent)

	var oldSum string
	if _, oldSum, err = readFileWithSha("versions.go"); err != nil {
		return
	}

	if newSum == oldSum {
		fmt.Println("no changes to file, exiting")
		return
	}

	if err = os.WriteFile("versions.go", wasmExecContent, 0644); err != nil {
		return
	}

	var repo *git.Repository
	if repo, err = git.PlainOpen("."); err != nil {
		return
	}
	var worktree *git.Worktree
	if worktree, err = repo.Worktree(); err != nil {
		return
	}
	if _, err = worktree.Add("versions.go"); err != nil {
		return
	}

	var newTag string
	if newTag, err = incrementMinor(); err != nil {
		return
	}
	var head *plumbing.Reference
	if head, err = repo.Head(); err != nil {
		return
	}

	opts := &git.CreateTagOptions{Message: newTag}
	if _, err = repo.CreateTag(newTag, head.Hash(), opts); err != nil {
		return
	}

	_, err = worktree.Commit("github actions update", &git.CommitOptions{
		Author: &object.Signature{Name: "mlctrez", Email: "mlctrez@gmail.com", When: time.Now()},
	})

	err = repo.Push(&git.PushOptions{Auth: &http.BasicAuth{Username: os.Getenv("INPUT_GITHUB_TOKEN")}})
	if err != nil {
		return
	}

	return
}

type SourceFile struct {
	buf *bytes.Buffer
}

func (sf *SourceFile) line(t string) *SourceFile {
	_, _ = sf.buf.WriteString(t + "\n")
	return sf
}

func (sf *SourceFile) format() (content []byte, err error) {
	return format.Source(sf.buf.Bytes())
}

func buildWasmExec(shaMapping map[string]string, contentMapping map[string][]byte) (content []byte, err error) {

	sf := &SourceFile{&bytes.Buffer{}}

	sf.line("package wasmexec")
	sf.line("// generated from https://github.com/mlctrez/wasmexec").line("")
	sf.line("import (").line(`"fmt"`).line(`"runtime"`).line(")")

	sf.line("func Current() (content []byte, err error) {")
	sf.line("return Version(runtime.Version())")
	sf.line("}").line("")

	sf.line("func Version(version string) (content []byte, err error) {").line("")

	sf.line("if contentFunc, ok := versionMap[version]; ok {")
	sf.line("content = []byte(contentFunc())")
	sf.line("}else{")
	sf.line(`err = fmt.Errorf("unsupported version %q", version)`)
	sf.line("}").line("")

	sf.line("return")
	sf.line("}").line("")

	var cmKeys []string
	for s := range contentMapping {
		cmKeys = append(cmKeys, s)
	}
	sort.Strings(cmKeys)

	for _, k := range cmKeys {
		sf.line(fmt.Sprintf("const %s = %q", shortSha(k), contentMapping[k]))
	}

	var goVersions []string
	for k := range shaMapping {
		goVersions = append(goVersions, k)
	}
	sort.Strings(goVersions)

	sf.line("var versionMap = map[string]func()string{")
	for _, goVersion := range goVersions {
		sf.line(fmt.Sprintf("%q: func() string { return %s },", goVersion, shortSha(shaMapping[goVersion])))
	}
	sf.line("}")

	return sf.format()
}

func shortSha(in string) string {
	return "sha" + strings.ToUpper(in[0:4]+in[64-4:])
}

func readFileWithSha(path string) (contents []byte, sum string, err error) {

	if contents, err = os.ReadFile(path); err != nil {
		return
	}

	sum = shaContents(contents)

	return
}

func shaContents(contents []byte) (sum string) {
	h := sha256.New()
	h.Write(contents)
	bs := h.Sum(nil)
	sum = fmt.Sprintf("%x", bs)
	return
}
