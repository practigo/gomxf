package gomxf

import (
	"bytes"
	"encoding/binary"
	"io"
)

var (
	// KeyFillItem is for items where the value is comprised of null or meaningless data.
	KeyFillItem = [16]byte{0x06, 0x0e, 0x2b, 0x34, 0x01, 0x01, 0x01, 0x02,
		0x03, 0x01, 0x02, 0x10, 0x01, 0x00, 0x00, 0x00}
	// KeyPartitionPack defines a Partition Pack, [13] is pack kind (0x02 ~ 0x04).
	KeyPartitionPack = [13]byte{0x06, 0x0e, 0x2b, 0x34, 0x02, 0x05, 0x01, 0x01,
		0x0d, 0x01, 0x02, 0x01, 0x01}
	// KeyOP is for Operational Patterns.
	KeyOP = [12]byte{0x06, 0x0e, 0x2b, 0x34, 0x04, 0x01, 0x01, 0x01,
		0x0d, 0x01, 0x02, 0x01}
)

const (
	HeaderPartitionPack = "HeaderPartitionPack"
	BodyPartitionPack   = "BodyPartitionPack"
	FooterPartitionPack = "FooterPartitionPack"
)

// IsPartitionPack checks if a key is KeyPartitionPack.
func IsPartitionPack(key []byte) bool {
	return bytes.Equal(key[:13], KeyPartitionPack[:]) && key[13] >= 2 && key[13] <= 4
}

// OperationalPattern defines how the MXF File config.
type OperationalPattern struct {
	Valid bool // an empty OP would be invalid
	//
	ItemComplexity    string
	PackageComplexity string
	// qualifier, true for 1
	External      bool
	NonStreamFile bool
	MultiTrack    bool
}

func parseOP(bs []byte) OperationalPattern {
	op := OperationalPattern{
		Valid:         true,
		External:      bs[14]&0b10 > 0,
		NonStreamFile: bs[14]&0b100 > 0,
		MultiTrack:    bs[14]&0b1000 > 0,
	}
	switch bs[12] {
	case 0x01:
		op.ItemComplexity = "Single"
	case 0x02:
		op.ItemComplexity = "Play-list"
	case 0x03:
		op.ItemComplexity = "Edit"
	}
	switch bs[13] {
	case 0x01:
		op.PackageComplexity = "Single"
	case 0x02:
		op.PackageComplexity = "Ganged"
	case 0x03:
		op.PackageComplexity = "Alternate"
	}
	return op
}

type packMeta struct {
	KAGSize           uint32
	ThisPartition     uint64
	PreviousPartition uint64
	FooterPartition   uint64
	HeaderByteCount   uint64
	IndexByteCount    uint64
	IndexSID          uint32
	BodyOffset        uint64
	BodySID           uint32
}

type pack struct {
	Kind      string
	Closed    bool
	Completed bool
	Meta      packMeta
	OP        OperationalPattern
	Essences  Batch
}

func decodePack(r io.ReaderAt, klv *KLV) (*pack, error) {
	k := klv.Key
	p := pack{
		Closed:    k[14] == 0x02 || k[14] == 0x04,
		Completed: k[14] == 0x03 || k[14] == 0x04,
	}
	switch k[13] {
	case 0x02:
		p.Kind = HeaderPartitionPack
	case 0x03:
		p.Kind = BodyPartitionPack
	case 0x04:
		p.Kind = FooterPartitionPack
	default:
		return &p, nil
	}
	bs, err := readData(r, klv)
	if err != nil {
		return nil, err
	}
	p.Meta.KAGSize = binary.BigEndian.Uint32(bs[4:8])
	p.Meta.ThisPartition = binary.BigEndian.Uint64(bs[8:16])
	p.Meta.PreviousPartition = binary.BigEndian.Uint64(bs[16:24])
	p.Meta.FooterPartition = binary.BigEndian.Uint64(bs[24:32])
	p.Meta.HeaderByteCount = binary.BigEndian.Uint64(bs[32:40])
	p.Meta.IndexByteCount = binary.BigEndian.Uint64(bs[40:48])
	p.Meta.IndexSID = binary.BigEndian.Uint32(bs[48:52])
	p.Meta.BodyOffset = binary.BigEndian.Uint64(bs[52:60])
	p.Meta.BodySID = binary.BigEndian.Uint32(bs[60:64])
	if bytes.Equal(KeyOP[:], bs[64:76]) {
		p.OP = parseOP(bs[64:80])
	}
	p.Essences = ParseBatch(bs[80:])
	return &p, nil
}

// Partition is a logical separation of a MXF file.
type Partition struct {
	pack
	Sub KLVs
}

// File represents a MXF file.
type File struct {
	Header *Partition
	Body   []*Partition
	Footer *Partition
}
