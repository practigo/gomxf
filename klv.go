package gomxf

import (
	"io"
	"os"
)

const (
	// MXF files shall have KLV Keys and ULs that are 16 bytes long.
	KeyLenMXF = int64(16)
)

// Range is the address of some continuous bytes.
type Range struct {
	Start int64
	End   int64
}

// KLV is the Key-Length-Value standard encoded data index.
// The value itself is not store here.
/*
| Key | BER |  [Value]  |
At        ValueStart
            |- Length ->| ==> ValueRange
|------- Size --------->|
*/
type KLV struct {
	Key    []byte
	Length int64
	// representive and redundant
	At         int64
	ValueStart int64
}

func (k *KLV) ValueRange() Range {
	return Range{
		Start: k.ValueStart,
		End:   k.ValueStart + k.Length,
	}
}

func (k *KLV) Size() int64 {
	return k.ValueStart + k.Length - k.At
}

// KLVs are a bunch of KLV that make up a MXF file.
type KLVs []*KLV

// BERLength parses the length according to BER,
// and returns also the offset of this length bytes.
func BERLength(data []byte) (l int64, offset int64) {
	octet1 := data[0]
	rest := int64(octet1 & 127)

	if octet1&128 == 0 {
		// Definite, short
		return rest, 1
	}

	// firstBit == 1
	if rest == 0 {
		panic("BER indefinite length occur")
	}
	if rest == 127 {
		panic("BER reserved length occur")
	}

	// rest ~ [1, 126]
	offset = 1
	for {
		l += int64(data[offset])
		offset += 1
		if offset > rest {
			break
		}
		l = l << 8
	}

	return l, offset
}

// ReadKLV reads a single KLV from the byte address at,
// where max should be the total size of the file to avoid invalid read.
func ReadKLV(r io.ReaderAt, at, max int64) (*KLV, error) {
	key := make([]byte, KeyLenMXF)
	_, err := r.ReadAt(key, at)
	if err != nil {
		return nil, err
	}
	klv := &KLV{
		Key: key,
		At:  at,
	}
	at += KeyLenMXF

	// MXF encoders shall not use long-form coding that exceeds a 9-byte BER encoded length
	n := int64(16)
	if at+n >= max {
		n = max - at
	}
	buf := make([]byte, n)
	_, err = r.ReadAt(buf, at)
	if err != nil {
		return nil, err
	}
	l, offset := BERLength(buf)
	klv.ValueStart = klv.At + KeyLenMXF + offset
	klv.Length = l

	return klv, nil
}

// Reader ...
type Reader struct {
	r    io.ReaderAt
	size int64
}

// Read reads at most n KLV elements from the file.
func (r *Reader) Read(n int) (KLVs, error) {
	at := int64(0)
	ks := make([]*KLV, 0)
	count := 0
	for {
		klv, err := ReadKLV(r.r, at, r.size)
		if err != nil {
			return nil, err
		}
		ks = append(ks, klv)
		count++
		at += klv.Size()
		if at >= r.size || (n > 0 && count >= n) {
			// fmt.Println(at, r.size)
			break
		}
	}
	return ks, nil
}

// NewReader opens the filename MXF and returns a Reader.
func NewReader(filename string) (*Reader, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	return &Reader{
		r:    f,
		size: info.Size(),
	}, nil
}
