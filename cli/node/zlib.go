package node

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io/ioutil"
)

func ZLibCompress(data []byte) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	zlibWriter := zlib.NewWriter(buf)
	_, err := zlibWriter.Write(data)
	if err != nil {
		return nil, fmt.Errorf("zlibWriter.Write error %s", err)
	}
	zlibWriter.Close()
	return buf.Bytes(), nil
}

func ZLibUncompress(data []byte) ([]byte, error) {
	buf := bytes.NewReader(data)
	zlibReader, err := zlib.NewReader(buf)
	if err != nil {
		return nil, fmt.Errorf("zlib.NewReader error %s", err)
	}
	defer zlibReader.Close()

	return ioutil.ReadAll(zlibReader)
}
