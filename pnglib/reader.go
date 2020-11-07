package pnglib

import (
	"bufio"
	"bytes"
	"os"
)

// PreProcessImage reads file to buffer
func PreProcessImage(dat *os.File) (*bytes.Reader, error) {
	stats, err := dat.Stat()
	if err != nil {
		return nil, err
	}

	size := stats.Size()
	b := make([]byte, size)

	bufR := bufio.NewReader(dat)
	_, err = bufR.Read(b)
	if err != nil {
		return nil, err
	}
	bReader := bytes.NewReader(b)

	return bReader, nil
}
