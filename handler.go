package gomxf

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Config is to customize how to parse the MXF file and show the info.
type Config struct {
	NRead       int
	Level       int
	Max         int
	ShowUnKnown bool
	KOI         string
}

func byteString(bs []byte, format string, sep string) string {
	ks := make([]string, 0)
	for _, b := range bs {
		ks = append(ks, fmt.Sprintf(format, b))
	}
	return strings.Join(ks, sep)
}

func showPartition(p *Partition, cfg *Config, showMore bool) {
	fmt.Printf("== %s: closed=%v, completed=%v, nContainers: %d, meta: %+v\n",
		p.Kind, p.Closed, p.Completed, p.Essences.N, p.Meta)
	if showMore {
		fmt.Printf("OP: %+v\n", p.OP)
		for i, e := range p.Essences.Elements {
			fmt.Printf("Essence#%d: %s\n", i, byteString(e, "%02x", "."))
		}
	}

	fmt.Printf("|-- total %d valid KLVs\n", len(p.Sub))

	if cfg.Level > 2 {
		for i, k := range p.Sub {
			if k.Name == KLVUnknown && !cfg.ShowUnKnown {
				continue
			}
			if cfg.Max > 0 && i > cfg.Max {
				fmt.Printf("... #%d~%d KLVs folded ...\n", i, len(p.Sub))
				break
			}
			name := k.Name
			if name == KLVUnknown {
				name = fmt.Sprintf("<[Unknown %s]>", byteString(k.Key, "%02x", "."))
			}
			fmt.Printf("#%d klv %s @%d with size %d: data-len: %d @%d\n",
				i, name, k.At, k.Size(), k.Length, k.ValueStart)
		}
	}
}

func parseCharNum(v string) (byte, int) {
	r := v[0]
	if len(v) > 1 {
		i, err := strconv.Atoi(v[1:])
		if err == nil {
			return r, i
		}
	}
	return r, -1
}

func showDetail(koi string, f *File, r io.ReaderAt) error {
	koi = strings.ToLower(koi)
	ps := strings.Split(koi, ":")
	if len(ps) < 2 {
		return errors.New("invalid KOI " + koi)
	}

	var klv *KLV

	part, bi := parseCharNum(ps[0])
	j, err := strconv.Atoi(ps[1])
	if err != nil {
		return err
	}

	style := byte('a') // auto
	limit := 128
	if len(ps) > 2 {
		style, limit = parseCharNum(ps[2])
	}

	switch part {
	case 'h':
		fmt.Printf("\nDetailed KLV in Header %d\n", j)
		klv = f.Header.Sub[j]
	case 'f':
		fmt.Printf("\nDetailed KLV in Footer %d\n", j)
		klv = f.Footer.Sub[j]
	case 'b':
		if bi < 0 || bi >= len(f.Body) {
			bi = 0
		}
		fmt.Printf("\nDetailed KLV in Body#%d %d\n", bi, j)
		klv = f.Body[bi].Sub[j]
	default:
		return errors.New("invalid KOI partition " + koi)
	}

	data, err := readData(r, klv)
	if err != nil {
		return err
	}

	if isStructuralMeta(klv.Key) && (style == 'a' || style == 's') {
		fmt.Println("==== Local Sets:")
		sets := ParseLocalSets(data, 2)
		for _, s := range sets {
			fmt.Printf("Tag: %04x, len %d, value: %s\n",
				s.Tag, s.Len, byteString(s.Value, "%02x", "."))
		}
	} else {
		// raw format
		n := len(data)
		i := 0
		fmt.Printf("==== Raw Data bytes with len %d (limit %d):\n", n, limit)
		for {
			next := i + 8
			if next > n {
				next = n // clip
			}
			fmt.Println(fmt.Sprintf("[%05d-%05d] ", i, next-1) + byteString(data[i:next], "0x%02x", ", "))
			if next >= n || (limit > 0 && next >= limit) {
				break
			}
			i = next
		}
	}

	return nil
}

// Parse parses the MXF file according to the config.
func Parse(filename string, cfg *Config) error {
	r, err := NewReader(filename)
	if err != nil {
		return err
	}

	ks, err := r.Read(cfg.NRead)
	if err != nil {
		return err
	}

	// level 1
	fmt.Printf("total %d klv read from file %s with size %d bytes\n", len(ks), filename, r.size)
	if cfg.Level < 2 {
		for i, k := range ks {
			fmt.Printf("#%d klv @%d with size %d: data-len: %d @%d\n",
				i, k.At, k.Size(), k.Length, k.ValueStart)
			if i >= cfg.Max-1 {
				break
			}
		}
		return nil
	}

	// level 2
	file, err := Decode(r.r, ks)
	if err != nil {
		return err
	}
	showPartition(file.Header, cfg, true)
	for _, b := range file.Body {
		showPartition(b, cfg, false)
	}
	if file.Footer != nil {
		showPartition(file.Footer, cfg, false)
	}

	if cfg.KOI != "" {
		return showDetail(cfg.KOI, file, r.r)
	}

	return nil
}
