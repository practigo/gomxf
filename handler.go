package gomxf

import (
	"fmt"
	"strings"
)

func keyString(bs []byte) string {
	ks := make([]string, 0)
	for _, b := range bs {
		ks = append(ks, fmt.Sprintf("%02x", b))
	}
	return strings.Join(ks, ".")
}

func show(k *KLV) string {
	return fmt.Sprintf("klv @%d: %s, size: %d, data-len: %d @%d",
		k.At, keyString(k.Key), k.Size(), k.Length, k.ValueStart)
}

// View ...
func View(filename string, n int) error {
	r, err := NewReader(filename)
	if err != nil {
		return err
	}

	ks, err := r.Read(n)
	if err != nil {
		return err
	}

	for i, k := range ks {
		fmt.Println(i, show(k))
	}

	return nil
}
