package gomxf

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

var (
	// Structural Metadata Sets, [5] is xx for length (0xff as placeholder)
	KeyStructural = [13]byte{0x06, 0x0e, 0x2b, 0x34, 0x02, 0xff, 0x01, 0x01,
		0x0d, 0x01, 0x01, 0x01, 0x01}
	// Primer Set, [14] is Version of the Primer Pack
	KeyPrimer = [14]byte{0x06, 0x0e, 0x2b, 0x34, 0x02, 0x05, 0x01, 0x01,
		0x0d, 0x01, 0x02, 0x01, 0x01, 0x05}
	// TODO: ? from ffmpeg mxfdec
	KeyEssenceElement = [12]byte{0x06, 0x0e, 0x2b, 0x34, 0x01, 0x02, 0x01, 0x01,
		0x0d, 0x01, 0x03, 0x01}
	// Index Table Segment, [5] is xx for length (0xff as placeholder)
	KeyIndexTable = [15]byte{0x06, 0x0e, 0x2b, 0x34, 0x02, 0xff, 0x01, 0x01,
		0x0d, 0x01, 0x02, 0x01, 0x01, 0x10, 0x01}
	// Random Index Pack
	KeyRIP = [15]byte{0x06, 0x0e, 0x2b, 0x34, 0x02, 0x05, 0x01, 0x01,
		0x0d, 0x01, 0x02, 0x01, 0x01, 0x11, 0x01}
	// some known structural metadata; TODO: make it external & configurable
	structuralMap = map[uint16]string{
		0x012f: "Preface",
		0x0130: "Identification",
		0x0118: "Content Storage",
		0x0123: "Essence Container Data",
		0x0136: "Material Package",
		0x0137: "Source Package",
		0x013b: "Timeline Track",
		0x010f: "Sequence",
		0x0111: "Source Clip",
		0x0114: "Timecode Component",
		0x0144: "Multiple Descriptor",
	}
	// KLVUnknown is for the unknown keys.
	KLVUnknown = "<[Unknown]>"
)

func isEssenceElement(key []byte) bool {
	return bytes.Equal(key[:12], KeyEssenceElement[:])
}

func isStructuralMeta(key []byte) bool {
	return bytes.Equal(key[:5], KeyStructural[:5]) && bytes.Equal(key[6:13], KeyStructural[6:])
}

func isIndexTable(key []byte) bool {
	return bytes.Equal(key[:5], KeyIndexTable[:5]) && bytes.Equal(key[6:15], KeyIndexTable[6:])
}

func recognizeKey(k []byte) string {
	if bytes.Equal(k[:14], KeyPrimer[:]) {
		return "Primer Pack"
	}
	if isStructuralMeta(k) {
		k14_15 := binary.BigEndian.Uint16(k[13:15])
		if name, ok := structuralMap[k14_15]; ok {
			return name
		}
		return fmt.Sprintf("Structural Meta 0x%04x", k14_15)
	}
	if isEssenceElement(k) {
		return "Essence Element"
	}
	if isIndexTable(k) {
		return "Index Table"
	}

	return KLVUnknown
}
