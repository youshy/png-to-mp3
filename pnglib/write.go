package pnglib

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
)

//WriteData writes new data to offset
func WriteData(r *bytes.Reader, c *CmdLineOpts, b []byte) error {
	offset, err := strconv.ParseInt(c.Offset, 10, 64)
	if err != nil {
		return err
	}

	w, err := os.OpenFile(c.Output, os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		return errors.New("Problem writing to the output file")
	}
	r.Seek(0, 0)

	buff := make([]byte, offset)
	r.Read(buff)
	w.Write(buff)
	w.Write(b)
	if c.Decode {
		r.Seek(int64(len(b)), 1) // right bitshift to overwrite encode chunk
	}
	_, err = io.Copy(w, r)
	if err != nil {
		return err
	}
	fmt.Printf("Success: %s created\n", c.Output)
	return nil
}
