package gomxf

import (
	"bytes"
	"errors"
	"io"
)

// Decode decodes the KLVs obtained from the Reader into a MXF File.
func Decode(r io.ReaderAt, ks KLVs) (*File, error) {
	f := File{}
	curPartition := &Partition{} // empty
	for _, k := range ks {
		if bytes.Equal(k.Key, KeyFillItem[:]) {
			continue
		}
		if IsPartitionPack(k.Key) {
			p, err := decodePack(r, k)
			if err != nil {
				return nil, err
			}
			if p.Kind == HeaderPartitionPack {
				if curPartition.Kind != "" { // should be empty
					return nil, errors.New("header partition should be the first")
				}
				if f.Header != nil {
					return nil, errors.New("repeated header partition")
				}
			} else {
				switch curPartition.Kind {
				case HeaderPartitionPack:
					f.Header = curPartition
				case BodyPartitionPack:
					f.Body = append(f.Body, curPartition)
				default:
					return nil, errors.New("invalid partitions order")
				}
			}
			curPartition = &Partition{
				pack: *p,
				Sub:  make(KLVs, 0),
			}
		} else {
			k.Name = recognizeKey(k.Key)
			curPartition.Sub = append(curPartition.Sub, k)
		}
	}
	// last one
	switch curPartition.Kind {
	case HeaderPartitionPack:
		f.Header = curPartition
	case BodyPartitionPack:
		f.Body = append(f.Body, curPartition)
	default:
		f.Footer = curPartition
	}
	return &f, nil
}
