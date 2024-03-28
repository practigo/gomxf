package gomxf

import (
	"fmt"
	"io"
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
	ROI         int
	ShowRaw     bool
	AsSets      bool
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

func getLine(bs []byte) string {
	ret := make([]string, 0)
	for _, b := range bs {
		ret = append(ret, fmt.Sprintf("0x%02x,", b))
	}
	return strings.Join(ret, " ")
}

func showKLV(r io.ReaderAt, k *KLV, l int, cfg *Config) error {
	fmt.Printf("klv @%d with size %d: data-len: %d @%d\n\n",
		k.At, k.Size(), k.Length, k.ValueStart)

	if !(cfg.ShowRaw || cfg.AsSets) {
		return nil
	}

	bs, err := readData(r, k)
	if err != nil {
		return err
	}

	if cfg.ShowRaw {
		n := len(bs)
		i := 0
		fmt.Println("== data: []byte{")
		for {
			next := i + l
			if next > n {
				next = n
			}
			fmt.Println(getLine(bs[i:next]))
			if next == n {
				break
			}
			i = next
			// line := bs[i:next]
		}
		fmt.Println("}")
	}

	if cfg.AsSets {
		fmt.Println("== Local Sets:")
		sets := ParseLocalSets(bs)
		for _, s := range sets {
			fmt.Println(s.View())
		}
	}

	return nil
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

	if cfg.ROI >= 0 {
		return showKLV(r.r, ks[cfg.ROI], 8, cfg)
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
