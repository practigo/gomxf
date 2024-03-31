package gomxf

import (
	"encoding/binary"
)

// Batch is a compound type comprising multiple individual elements where the elements
// are unordered, the type is defined, the count of items is explicit and the size of each
// item is fixed. The Batch consists of a header of 8 bytes followed by the Batch elements.
// The first 4 bytes of the header define the number of elements in the Batch.
// The last 4 bytes of the header define the length of each element.
type Batch struct {
	N        int
	Len      int
	Elements [][]byte
}

// ParseBatch parses the bytes into a Batch with the elements untouched.
func ParseBatch(data []byte) (b Batch) {
	b.N = int(binary.BigEndian.Uint32(data[:4]))
	b.Len = int(binary.BigEndian.Uint32(data[4:8]))
	offset := 8
	for i := 0; i < b.N; i++ {
		b.Elements = append(b.Elements, data[offset:offset+b.Len])
		offset += b.Len
	}
	return b
}

// LocalSet is a set where each Item is encoded using a locally unique tag value of the
// same length. An MXF Set employing 2-byte Local Tag encoding, and either 2-byte or
// BER length.
type LocalSet struct {
	Tag   uint16
	Len   int
	Value []byte
}

// ParseLocalSets parses the bytes into some LocalSet with lenBytes length.
// TODO: support BER length.
func ParseLocalSets(bs []byte, lenBytes int) []LocalSet {
	if lenBytes < 0 {
		panic("BER length not supported yet")
	}
	i := 0
	n := len(bs)
	ret := make([]LocalSet, 0)
	for {
		if n-i < 2+lenBytes {
			break
		}
		l := int(binary.BigEndian.Uint16(bs[i+2 : i+2+lenBytes]))
		ret = append(ret, LocalSet{
			Tag:   binary.BigEndian.Uint16(bs[i : i+2]),
			Len:   l,
			Value: bs[i+2+lenBytes : i+2+lenBytes+l],
		})

		i += (4 + l)
	}
	return ret
}
