package sourcefile

import (
	"bytes"
	"go/format"
)

type SourceFile struct {
	buf *bytes.Buffer
}

func New() *SourceFile {
	return &SourceFile{buf: &bytes.Buffer{}}
}

func (sf *SourceFile) L(t string) *SourceFile {
	_, _ = sf.buf.WriteString(t + "\n")
	return sf
}

func (sf *SourceFile) Format() (content []byte, err error) {
	return format.Source(sf.buf.Bytes())
}
