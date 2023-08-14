package wasmexec

import (
	"bytes"
	"compress/zlib"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"runtime"
	"sync"
)

var mu = &sync.Mutex{}
var cached map[string][]byte

func Current() (content []byte, err error) {
	return Version(runtime.Version())
}

func Version(version string) (contents []byte, err error) {
	mu.Lock()
	defer mu.Unlock()

	if cached == nil {
		cached = make(map[string][]byte)
	}
	var data []byte
	if data, err = readContents(version); err != nil {
		return nil, err
	}
	cached[version] = data
	return data, nil
}

func readContents(version string) (contents []byte, err error) {
	wantedSha := TagToSha(version)
	if wantedSha == "" {
		return nil, fmt.Errorf("unsupported version %q", version)
	}

	var total uint32
	var sha = make([]byte, 32)
	var length int64
	var read int

	var reader io.ReadCloser
	reader, err = zlib.NewReader(bytes.NewBuffer(compressed))

	if err = binary.Read(reader, binary.BigEndian, &total); err != nil {
		return nil, err
	}

	var ti uint32
	for ti = 0; ti < total; ti++ {
		if read, err = reader.Read(sha); err != nil {
			return nil, err
		}
		if read != 32 {
			return nil, fmt.Errorf("unable to read full sha")
		}
		if err = binary.Read(reader, binary.BigEndian, &length); err != nil {
			return nil, err
		}

		contents = make([]byte, length)
		if read, err = io.ReadFull(reader, contents); err != nil {
			return nil, err
		}
		if read != int(length) {
			return nil, fmt.Errorf("unable to read full content, expected %d but read only %d", length, read)
		}

		if wantedSha == shaString(contents) {
			return contents, nil
		}

	}

	read, err = reader.Read([]byte{0})
	if read != 0 || err != io.EOF {
		return nil, fmt.Errorf("data did not end conrrectly")
	}

	return nil, fmt.Errorf("unable to match sha %q", wantedSha)

}

func shaString(contents []byte) (sum string) {
	return fmt.Sprintf("%x", shaByte(contents))
}

func shaByte(contents []byte) []byte {
	h := sha256.New()
	h.Write(contents)
	bs := h.Sum(nil)
	return bs
}
