package gitver

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"github.com/mlctrez/wasmexec/shautil"
	"github.com/mlctrez/wasmexec/sourcefile"
	"github.com/pkg/errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
)

type GitVer interface {
	Run() error
	Versions(sf *sourcefile.SourceFile) error
}

func New(repository, tempDir string, paths []string, tagPrefix string) GitVer {
	return &gitVer{repository: repository, tempDir: tempDir, paths: paths, tagPrefix: tagPrefix}
}

type gitVer struct {
	repository string
	tempDir    string
	paths      []string
	tagPrefix  string

	tags         []string
	tagMapping   map[string]string
	shaToContent map[string][]byte
	compressed   []byte
}

func (g *gitVer) Run() error {
	var err error
	steps := []func() error{
		g.reset,
		g.clone,
		g.getTags,
		g.getMappings,
		g.compress,
	}
	for i, part := range steps {
		name := runtime.FuncForPC(reflect.ValueOf(part).Pointer()).Name()
		fmt.Printf("%02d %s\n", i, name)
		if err = part(); err != nil {
			return err
		}
	}
	return nil
}

func (g *gitVer) reset() error {
	g.tags = []string{}
	g.tagMapping = map[string]string{}
	g.shaToContent = map[string][]byte{}

	return nil
}

func (g *gitVer) clone() error {
	var err error
	if _, err = os.Stat(g.tempDir); os.IsNotExist(err) {
		command := exec.Command("git", "clone", "-q", g.repository, g.tempDir)
		err = command.Run()
	}
	return err
}

func (g *gitVer) getTags() error {
	command := exec.Command("git", "tag", "-l")
	command.Dir = g.tempDir

	var err error
	var output []byte
	if output, err = command.CombinedOutput(); err != nil {
		return errors.WithMessage(err, string(output))
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		tag := scanner.Text()
		if strings.HasPrefix(tag, g.tagPrefix) {
			g.tags = append(g.tags, tag)
		}
	}
	sort.Strings(g.tags)
	return nil
}

func (g *gitVer) getMappings() error {
	for _, tag := range g.tags {
		for _, path := range g.paths {

			command := exec.Command("git", "checkout", tag, path)
			command.Dir = g.tempDir

			var err error
			var output []byte
			if output, err = command.CombinedOutput(); err != nil {
				outputMessage := string(output)
				if strings.Contains(outputMessage, "did not match any file(s) known to git") {
					continue
				}
				return errors.WithMessage(err, outputMessage)
			}

			var content []byte
			if content, err = os.ReadFile(filepath.Join(g.tempDir, path)); err != nil {
				return err
			}

			sha := shautil.ShaString(content)
			g.shaToContent[sha] = content
			g.tagMapping[tag] = sha
		}

	}

	return nil
}

func (g *gitVer) compress() error {
	buf := &bytes.Buffer{}

	var err error
	var writer *zlib.Writer
	if writer, err = zlib.NewWriterLevel(buf, zlib.BestCompression); err != nil {
		return err
	}

	var shaKeys []string
	for k := range g.shaToContent {
		shaKeys = append(shaKeys, k)
	}
	sort.Strings(shaKeys)

	if err = binary.Write(writer, binary.BigEndian, uint32(len(g.shaToContent))); err != nil {
		return err
	}

	for _, key := range shaKeys {
		content := g.shaToContent[key]
		contentLength := int64(len(content))
		shaBytes := shautil.ShaBytes(content)
		if _, err = writer.Write(shaBytes); err != nil {
			return err
		}
		if err = binary.Write(writer, binary.BigEndian, contentLength); err != nil {
			return err
		}
		if _, err = writer.Write(content); err != nil {
			return err
		}
	}
	if err = writer.Flush(); err != nil {
		return err
	}
	if err = writer.Close(); err != nil {
		return err
	}
	g.compressed = buf.Bytes()

	return err
}

func (g *gitVer) tagToSha(sf *sourcefile.SourceFile) {
	sf.L("var tagToShaMap = map[string]string{")

	for _, tag := range g.tags {
		if sha, ok := g.tagMapping[tag]; ok {
			sf.L(fmt.Sprintf("%q:%q,", tag, sha))
		}
	}
	sf.L("}").L("")

	sf.L("func TagToSha(tag string) string {")
	sf.L("return tagToShaMap[tag]")
	sf.L("}")
}

func (g *gitVer) compressedVar(sf *sourcefile.SourceFile) error {
	sf.L(fmt.Sprintf("// compressed length %d", len(g.compressed)))
	sf.L("var compressed = []byte{")

	buf := bytes.NewBuffer(g.compressed)

	lineBuf := make([]byte, 18)
	var read int
	var err error
	if read, err = buf.Read(lineBuf); err != nil {
		return err
	}
	for read > 0 {
		var line string
		for i := 0; i < read; i++ {
			line += fmt.Sprintf("0x%02x,", lineBuf[i])
		}
		sf.L(line)
		read, err = buf.Read(lineBuf)
		if read == 0 || err == io.EOF {
			break
		}
	}
	if err != nil && err != io.EOF {
		return err
	}

	sf.L("}")
	return nil
}

func (g *gitVer) Versions(sf *sourcefile.SourceFile) error {
	sf.L("package wasmexec")
	g.tagToSha(sf)
	var err error
	if err = g.compressedVar(sf); err != nil {
		return err
	}

	return nil
}
