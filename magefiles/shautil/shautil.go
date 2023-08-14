package shautil

import (
	"crypto/sha256"
	"fmt"
	"os"
)

func ReadWithSha(path string) (contents []byte, sum string, err error) {
	if contents, err = os.ReadFile(path); err != nil {
		return nil, "", err
	}
	return contents, ShaString(contents), nil
}

func ShaString(contents []byte) (sum string) {
	return fmt.Sprintf("%x", ShaBytes(contents))
}

func ShaBytes(contents []byte) (sha []byte) {
	h := sha256.New()
	h.Write(contents)
	bs := h.Sum(nil)
	return bs
}
