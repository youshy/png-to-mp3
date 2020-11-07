package pnglib

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/crc32"
	"log"
	"strconv"
	"strings"
)

const (
	endChunkType = "IEND"
)

// Header holds the first UINT64 (Magic Bytes)
// Magic Bytes will allow us to validate png images
type Header struct {
	Header uint64
}

// Chunk represents a data byte chunk segment
type Chunk struct {
	Size uint32 // 4 bytes, defines size of the data
	Type uint32 // 4 bytes, defines type based on PNG spec
	Data []byte // unlimited, defined by Size
	CRC  uint32 // 4 bytes, checksum of combined Type and Data
}

// MetaChunk inherits a Chunk struct
type MetaChunk struct {
	Chk    Chunk
	Offset int64
}

// ProcessImage is a wrapper to parse PNG bytes
func (mc *MetaChunk) ProcessImage(b *bytes.Reader, c *CmdLineOpts) error {
	err := mc.validate(b)
	if err != nil {
		return err
	}
	// If no action provided
	if (c.Offset != "") && (c.Encode == false && c.Decode == false) {
		var m MetaChunk
		m.Chk.Data = []byte(c.Payload)
		m.Chk.Type = m.strToInt(c.Type)
		m.Chk.Size = m.createChunkSize()
		m.Chk.CRC = m.createChunkCRC()
		bm := m.marshalData()
		bmb := bm.Bytes()
		fmt.Printf("Payload Original: % X\n", []byte(c.Payload))
		fmt.Printf("Payload: % X\n", m.Chk.Data)
		WriteData(b, c, bmb)
	}
	if (c.Offset != "") && c.Encode {
		var m MetaChunk
		m.Chk.Data = XorEncode([]byte(c.Payload), c.Key)
		m.Chk.Type = m.strToInt(c.Type)
		m.Chk.Size = m.createChunkSize()
		m.Chk.CRC = m.createChunkCRC()
		bm := m.marshalData()
		bmb := bm.Bytes()
		fmt.Printf("Payload Original: % X\n", []byte(c.Payload))
		fmt.Printf("Payload Encode: % X\n", m.Chk.Data)
		WriteData(b, c, bmb)
	}
	if (c.Offset != "") && c.Decode {
		var m MetaChunk
		offset, _ := strconv.ParseInt(c.Offset, 10, 64)
		b.Seek(offset, 0)
		m.readChunk(b)
		origData := m.Chk.Data
		m.Chk.Data = XorDecode(m.Chk.Data, c.Key)
		m.Chk.CRC = m.createChunkCRC()
		bm := m.marshalData()
		bmb := bm.Bytes()
		fmt.Printf("Payload Original: % X\n", origData)
		fmt.Printf("Payload Decode: % X\n", m.Chk.Data)
		WriteData(b, c, bmb)
	}
	if c.Meta {
		count := 1 //Start at 1 because 0 is reserved for magic byte
		var chunkType string
		for chunkType != endChunkType {
			mc.getOffset(b)
			mc.readChunk(b)
			fmt.Println("---- Chunk # " + strconv.Itoa(count) + " ----")
			fmt.Printf("Chunk Offset: %#02x\n", mc.Offset)
			fmt.Printf("Chunk Length: %s bytes\n", strconv.Itoa(int(mc.Chk.Size)))
			fmt.Printf("Chunk Type: %s\n", mc.chunkTypeToString())
			fmt.Printf("Chunk Importance: %s\n", mc.checkCritType())
			if c.Suppress == false {
				fmt.Printf("Chunk Data: %#x\n", mc.Chk.Data)
			} else if c.Suppress {
				fmt.Printf("Chunk Data: %s\n", "Suppressed")
			}
			fmt.Printf("Chunk CRC: %x\n", mc.Chk.CRC)
			chunkType = mc.chunkTypeToString()
			count++
		}
	}

	return nil
}

func (mc *MetaChunk) marshalData() *bytes.Buffer {
	bytesMSB := new(bytes.Buffer)
	if err := binary.Write(bytesMSB, binary.BigEndian, mc.Chk.Size); err != nil {
		log.Fatal(err)
	}
	if err := binary.Write(bytesMSB, binary.BigEndian, mc.Chk.Type); err != nil {
		log.Fatal(err)
	}
	if err := binary.Write(bytesMSB, binary.BigEndian, mc.Chk.Data); err != nil {
		log.Fatal(err)
	}
	if err := binary.Write(bytesMSB, binary.BigEndian, mc.Chk.CRC); err != nil {
		log.Fatal(err)
	}

	return bytesMSB
}

// Validate checks if valid PNG
func (*MetaChunk) validate(b *bytes.Reader) error {
	var header Header
	if err := binary.Read(b, binary.BigEndian, &header.Header); err != nil {
		return err
	}

	bArr := make([]byte, 8)
	// Keep the bytes in order
	binary.BigEndian.PutUint64(bArr, header.Header)

	if string(bArr[1:4]) != "PNG" {
		return errors.New("Provided file is not a valid PNG format")
	}
	return nil
}

func (mc *MetaChunk) readChunk(b *bytes.Reader) {
	err := mc.readChunkSize(b)
	if err != nil {
		log.Fatal(err)
	}
	err = mc.readChunkType(b)
	if err != nil {
		log.Fatal(err)
	}
	err = mc.readChunkBytes(b)
	if err != nil {
		log.Fatal(err)
	}
	err = mc.readChunkCRC(b)
	if err != nil {
		log.Fatal(err)
	}
}

func (mc *MetaChunk) readChunkSize(b *bytes.Reader) error {
	if err := binary.Read(b, binary.BigEndian, &mc.Chk.Size); err != nil {
		return err
	}
	return nil
}

func (mc *MetaChunk) readChunkType(b *bytes.Reader) error {
	if err := binary.Read(b, binary.BigEndian, &mc.Chk.Type); err != nil {
		return err
	}
	return nil
}

func (mc *MetaChunk) readChunkBytes(b *bytes.Reader) error {
	mc.Chk.Data = make([]byte, mc.Chk.Size)
	if err := binary.Read(b, binary.BigEndian, &mc.Chk.Data); err != nil {
		return err
	}
	return nil
}

func (mc *MetaChunk) readChunkCRC(b *bytes.Reader) error {
	if err := binary.Read(b, binary.BigEndian, &mc.Chk.CRC); err != nil {
		return err
	}
	return nil
}

func (mc *MetaChunk) getOffset(b *bytes.Reader) {
	offset, _ := b.Seek(0, 1)
	mc.Offset = offset
}

func (mc *MetaChunk) chunkTypeToString() string {
	h := fmt.Sprintf("%x", mc.Chk.Type)
	decoded, _ := hex.DecodeString(h)
	return fmt.Sprintf("%s", decoded)
}

func (mc *MetaChunk) checkCritType() string {
	fChar := string([]rune(mc.chunkTypeToString())[0])
	if fChar == strings.ToUpper(fChar) {
		return "Critical"
	}
	return "Ancillary"
}

func (mc *MetaChunk) createChunkSize() uint32 {
	return uint32(len(mc.Chk.Data))
}

func (mc *MetaChunk) createChunkCRC() uint32 {
	bytesMSB := new(bytes.Buffer)
	if err := binary.Write(bytesMSB, binary.BigEndian, mc.Chk.Type); err != nil {
		log.Fatal(err)
	}
	if err := binary.Write(bytesMSB, binary.BigEndian, mc.Chk.Data); err != nil {
		log.Fatal(err)
	}
	return crc32.ChecksumIEEE(bytesMSB.Bytes())
}

func (mc *MetaChunk) strToInt(s string) uint32 {
	t := []byte(s)
	return binary.BigEndian.Uint32(t)
}
