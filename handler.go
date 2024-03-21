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

type Config struct {
	NRead       int
	ShowUnKnown bool
	ShowFill    bool
}

func show(i int, k *KLV, d KLVData, cfg *Config) {
	if d.Known() {
		if d.IsFill() {
			if cfg.ShowFill {
				fmt.Printf("klv#%d Fill Item with size %d\n", i, k.Size())
			}
		} else {
			fmt.Printf("klv#%d @%d with size %d: data-len: %d @%d\n== %s\n",
				i, k.At, k.Size(), k.Length, k.ValueStart, d.View())
		}
	} else {
		if cfg.ShowUnKnown {
			fmt.Printf("klv#%d @%d with size %d: unknown key %s\n",
				i, k.At, k.Size(), keyString(k.Key))
		}
	}
}

// View ...
func View(filename string, cfg *Config) error {
	r, err := NewReader(filename)
	if err != nil {
		return err
	}

	ks, err := r.Read(cfg.NRead)
	if err != nil {
		return err
	}

	ds, err := Decode4View(r.r, ks)
	if err != nil {
		return err
	}

	for i, d := range ds {
		show(i, ks[i], d, cfg)
	}

	return nil
}
