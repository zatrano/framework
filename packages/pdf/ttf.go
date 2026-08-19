package pdf

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// ttfFont is a minimally parsed TrueType face for PDF embedding.
type ttfFont struct {
	data       []byte
	postscript string
	unitsPerEm uint16
	ascent     int16
	descent    int16
	capHeight  int16
	bbox       [4]int16 // xMin, yMin, xMax, yMax
	numGlyphs  int
	advance    []uint16 // design units, index = glyph id
	cmap       map[rune]uint16
}

func needsUnicode(lines []string) bool {
	for _, line := range lines {
		for _, r := range line {
			if r > 127 {
				return true
			}
		}
	}
	return false
}

func loadSystemTTF() (*ttfFont, error) {
	for _, path := range systemFontCandidates() {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		f, err := parseTTF(raw)
		if err != nil {
			continue
		}
		if f.postscript == "" {
			f.postscript = fontBaseName(path)
		}
		return f, nil
	}
	return nil, fmt.Errorf("pdf: no system TrueType font found")
}

func fontBaseName(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	if ext != "" {
		base = base[:len(base)-len(ext)]
	}
	if base == "" {
		return "Embedded"
	}
	return sanitizePDFName(base)
}

func systemFontCandidates() []string {
	var out []string
	switch runtime.GOOS {
	case "windows":
		windir := os.Getenv("WINDIR")
		if windir == "" {
			windir = `C:\Windows`
		}
		fonts := filepath.Join(windir, "Fonts")
		out = append(out,
			filepath.Join(fonts, "arial.ttf"),
			filepath.Join(fonts, "arialuni.ttf"),
			filepath.Join(fonts, "calibri.ttf"),
			filepath.Join(fonts, "tahoma.ttf"),
			filepath.Join(fonts, "segoeui.ttf"),
		)
	case "darwin":
		out = append(out,
			"/System/Library/Fonts/Supplemental/Arial.ttf",
			"/Library/Fonts/Arial.ttf",
			"/System/Library/Fonts/Supplemental/Arial Unicode.ttf",
			"/Library/Fonts/Arial Unicode.ttf",
		)
	default:
		out = append(out,
			"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
			"/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf",
			"/usr/share/fonts/truetype/freefont/FreeSans.ttf",
			"/usr/share/fonts/TTF/DejaVuSans.ttf",
			"/usr/share/fonts/dejavu/DejaVuSans.ttf",
		)
	}
	return out
}

func parseTTF(data []byte) (*ttfFont, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("ttf: too short")
	}
	scaler := binary.BigEndian.Uint32(data[0:4])
	if scaler == 0x4F54544F { // OTTO / CFF
		return nil, fmt.Errorf("ttf: CFF/OpenType not supported")
	}
	numTables := int(binary.BigEndian.Uint16(data[4:6]))
	tables := map[string]ttfTable{}
	off := 12
	for i := 0; i < numTables; i++ {
		if off+16 > len(data) {
			return nil, fmt.Errorf("ttf: truncated table directory")
		}
		tag := string(data[off : off+4])
		offset := binary.BigEndian.Uint32(data[off+8 : off+12])
		length := binary.BigEndian.Uint32(data[off+12 : off+16])
		tables[tag] = ttfTable{offset: offset, length: length}
		off += 16
	}
	for _, tag := range []string{"cmap", "head", "hhea", "maxp", "hmtx"} {
		if _, ok := tables[tag]; !ok {
			return nil, fmt.Errorf("ttf: missing %s", tag)
		}
	}

	f := &ttfFont{data: data, cmap: map[rune]uint16{}}

	head, err := sliceTable(data, tables["head"])
	if err != nil {
		return nil, err
	}
	if len(head) < 54 {
		return nil, fmt.Errorf("ttf: short head")
	}
	f.unitsPerEm = binary.BigEndian.Uint16(head[18:20])
	f.bbox[0] = int16(binary.BigEndian.Uint16(head[36:38]))
	f.bbox[1] = int16(binary.BigEndian.Uint16(head[38:40]))
	f.bbox[2] = int16(binary.BigEndian.Uint16(head[40:42]))
	f.bbox[3] = int16(binary.BigEndian.Uint16(head[42:44]))

	hhea, err := sliceTable(data, tables["hhea"])
	if err != nil {
		return nil, err
	}
	if len(hhea) < 36 {
		return nil, fmt.Errorf("ttf: short hhea")
	}
	f.ascent = int16(binary.BigEndian.Uint16(hhea[4:6]))
	f.descent = int16(binary.BigEndian.Uint16(hhea[6:8]))
	numOfLongHorMetrics := int(binary.BigEndian.Uint16(hhea[34:36]))

	maxp, err := sliceTable(data, tables["maxp"])
	if err != nil {
		return nil, err
	}
	if len(maxp) < 6 {
		return nil, fmt.Errorf("ttf: short maxp")
	}
	f.numGlyphs = int(binary.BigEndian.Uint16(maxp[4:6]))

	hmtx, err := sliceTable(data, tables["hmtx"])
	if err != nil {
		return nil, err
	}
	f.advance = make([]uint16, f.numGlyphs)
	for i := 0; i < f.numGlyphs; i++ {
		var adv uint16
		if i < numOfLongHorMetrics {
			pos := i * 4
			if pos+2 > len(hmtx) {
				break
			}
			adv = binary.BigEndian.Uint16(hmtx[pos : pos+2])
		} else if numOfLongHorMetrics > 0 {
			pos := (numOfLongHorMetrics - 1) * 4
			adv = binary.BigEndian.Uint16(hmtx[pos : pos+2])
		}
		f.advance[i] = adv
	}

	cmapData, err := sliceTable(data, tables["cmap"])
	if err != nil {
		return nil, err
	}
	if err := parseCmap(cmapData, f.cmap); err != nil {
		return nil, err
	}

	f.capHeight = f.ascent
	if nameTbl, ok := tables["name"]; ok {
		if nameData, err := sliceTable(data, nameTbl); err == nil {
			if ps := readPostscriptName(nameData); ps != "" {
				f.postscript = sanitizePDFName(ps)
			}
		}
	}
	if f.postscript == "" {
		f.postscript = "Embedded"
	}
	return f, nil
}

type ttfTable struct {
	offset, length uint32
}

func sliceTable(data []byte, t ttfTable) ([]byte, error) {
	end := int(t.offset) + int(t.length)
	if int(t.offset) > len(data) || end > len(data) {
		return nil, fmt.Errorf("ttf: table out of range")
	}
	return data[t.offset:end], nil
}

func parseCmap(data []byte, out map[rune]uint16) error {
	if len(data) < 4 {
		return fmt.Errorf("ttf: short cmap")
	}
	numTables := int(binary.BigEndian.Uint16(data[2:4]))
	type rec struct {
		platform, encoding uint16
		offset             uint32
	}
	var bestFormat4, bestFormat12 *rec
	for i := 0; i < numTables; i++ {
		o := 4 + i*8
		if o+8 > len(data) {
			break
		}
		r := rec{
			platform: binary.BigEndian.Uint16(data[o : o+2]),
			encoding: binary.BigEndian.Uint16(data[o+2 : o+4]),
			offset:   binary.BigEndian.Uint32(data[o+4 : o+8]),
		}
		if int(r.offset)+2 > len(data) {
			continue
		}
		format := binary.BigEndian.Uint16(data[r.offset : r.offset+2])
		uni := r.platform == 0 || (r.platform == 3 && (r.encoding == 1 || r.encoding == 10))
		if !uni {
			continue
		}
		switch format {
		case 4:
			bestFormat4 = &r
		case 12:
			bestFormat12 = &r
		}
	}
	if bestFormat12 != nil {
		return parseCmapFormat12(data[bestFormat12.offset:], out)
	}
	if bestFormat4 != nil {
		return parseCmapFormat4(data[bestFormat4.offset:], out)
	}
	for i := 0; i < numTables; i++ {
		o := 4 + i*8
		if o+8 > len(data) {
			break
		}
		off := binary.BigEndian.Uint32(data[o+4 : o+8])
		if int(off)+2 > len(data) {
			continue
		}
		if binary.BigEndian.Uint16(data[off:off+2]) == 4 {
			return parseCmapFormat4(data[off:], out)
		}
	}
	return fmt.Errorf("ttf: no usable cmap")
}

func parseCmapFormat4(data []byte, out map[rune]uint16) error {
	if len(data) < 14 {
		return fmt.Errorf("ttf: short cmap4")
	}
	segCountX2 := int(binary.BigEndian.Uint16(data[6:8]))
	segCount := segCountX2 / 2
	endCount := 14
	startCount := endCount + segCountX2 + 2
	idDelta := startCount + segCountX2
	idRangeOffset := idDelta + segCountX2
	if idRangeOffset+segCountX2 > len(data) {
		return fmt.Errorf("ttf: cmap4 truncated")
	}
	for i := 0; i < segCount; i++ {
		end := binary.BigEndian.Uint16(data[endCount+2*i : endCount+2*i+2])
		start := binary.BigEndian.Uint16(data[startCount+2*i : startCount+2*i+2])
		delta := int16(binary.BigEndian.Uint16(data[idDelta+2*i : idDelta+2*i+2]))
		rangeOff := binary.BigEndian.Uint16(data[idRangeOffset+2*i : idRangeOffset+2*i+2])
		for c := int(start); c <= int(end); c++ {
			var gid uint16
			if rangeOff == 0 {
				gid = uint16(c + int(delta))
			} else {
				p := idRangeOffset + 2*i + int(rangeOff) + 2*(c-int(start))
				if p+2 > len(data) {
					continue
				}
				gid = binary.BigEndian.Uint16(data[p : p+2])
				if gid != 0 {
					gid = uint16(int(gid) + int(delta))
				}
			}
			if gid != 0 {
				out[rune(c)] = gid
			}
			if c == 0xFFFF {
				break
			}
		}
	}
	return nil
}

func parseCmapFormat12(data []byte, out map[rune]uint16) error {
	if len(data) < 16 {
		return fmt.Errorf("ttf: short cmap12")
	}
	nGroups := int(binary.BigEndian.Uint32(data[12:16]))
	off := 16
	for i := 0; i < nGroups; i++ {
		if off+12 > len(data) {
			break
		}
		start := binary.BigEndian.Uint32(data[off : off+4])
		end := binary.BigEndian.Uint32(data[off+4 : off+8])
		glyph := binary.BigEndian.Uint32(data[off+8 : off+12])
		off += 12
		for c, g := start, glyph; c <= end; c, g = c+1, g+1 {
			if g != 0 && g <= 0xFFFF {
				out[rune(c)] = uint16(g)
			}
		}
	}
	return nil
}

func readPostscriptName(nameData []byte) string {
	if len(nameData) < 6 {
		return ""
	}
	count := int(binary.BigEndian.Uint16(nameData[2:4]))
	storage := int(binary.BigEndian.Uint16(nameData[4:6]))
	var best string
	for i := 0; i < count; i++ {
		o := 6 + i*12
		if o+12 > len(nameData) {
			break
		}
		platform := binary.BigEndian.Uint16(nameData[o : o+2])
		encoding := binary.BigEndian.Uint16(nameData[o+2 : o+4])
		lang := binary.BigEndian.Uint16(nameData[o+4 : o+6])
		nameID := binary.BigEndian.Uint16(nameData[o+6 : o+8])
		length := int(binary.BigEndian.Uint16(nameData[o+8 : o+10]))
		offset := int(binary.BigEndian.Uint16(nameData[o+10 : o+12]))
		if nameID != 6 {
			continue
		}
		start := storage + offset
		end := start + length
		if start < 0 || end > len(nameData) {
			continue
		}
		raw := nameData[start:end]
		var s string
		if platform == 0 || (platform == 3 && encoding == 1) {
			var runes []rune
			for j := 0; j+1 < len(raw); j += 2 {
				r := rune(binary.BigEndian.Uint16(raw[j : j+2]))
				if r != 0 {
					runes = append(runes, r)
				}
			}
			s = string(runes)
		} else {
			s = string(raw)
		}
		if s == "" {
			continue
		}
		if platform == 3 && lang == 0x0409 {
			return s
		}
		best = s
	}
	return best
}

func sanitizePDFName(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '+' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" {
		return "Embedded"
	}
	return out
}

func (f *ttfFont) glyph(r rune) uint16 {
	if g, ok := f.cmap[r]; ok {
		return g
	}
	if g, ok := f.cmap['?']; ok {
		return g
	}
	return 0
}

func (f *ttfFont) widthPDF(gid uint16) int {
	if int(gid) >= len(f.advance) || f.unitsPerEm == 0 {
		return 500
	}
	return int(f.advance[gid]) * 1000 / int(f.unitsPerEm)
}

func (f *ttfFont) encodeLineHex(s string) (string, map[uint16]struct{}) {
	used := map[uint16]struct{}{}
	var b bytes.Buffer
	b.WriteByte('<')
	for _, r := range s {
		gid := f.glyph(r)
		used[gid] = struct{}{}
		fmt.Fprintf(&b, "%04X", gid)
	}
	b.WriteByte('>')
	return b.String(), used
}

func flateBytes(raw []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(raw); err != nil {
		_ = w.Close()
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func buildWidthArray(f *ttfFont, used map[uint16]struct{}) string {
	ids := make([]int, 0, len(used))
	for g := range used {
		ids = append(ids, int(g))
	}
	sort.Ints(ids)
	var b strings.Builder
	b.WriteString("[")
	for _, id := range ids {
		fmt.Fprintf(&b, "%d [%d] ", id, f.widthPDF(uint16(id)))
	}
	b.WriteString("]")
	return b.String()
}

func buildToUnicode(f *ttfFont, used map[uint16]struct{}) string {
	gidToRune := map[uint16]rune{}
	for r, g := range f.cmap {
		if _, ok := used[g]; !ok {
			continue
		}
		if _, exists := gidToRune[g]; !exists {
			gidToRune[g] = r
		}
	}
	ids := make([]int, 0, len(gidToRune))
	for g := range gidToRune {
		ids = append(ids, int(g))
	}
	sort.Ints(ids)

	var b strings.Builder
	b.WriteString("/CIDInit /ProcSet findresource begin\n")
	b.WriteString("12 dict begin\nbegincmap\n")
	b.WriteString("/CIDSystemInfo << /Registry (Adobe) /Ordering (UCS) /Supplement 0 >> def\n")
	b.WriteString("/CMapName /Adobe-Identity-UCS def\n")
	b.WriteString("/CMapType 2 def\n")
	b.WriteString("1 begincodespacerange\n<0000> <FFFF>\nendcodespacerange\n")
	const chunk = 100
	for i := 0; i < len(ids); i += chunk {
		end := i + chunk
		if end > len(ids) {
			end = len(ids)
		}
		fmt.Fprintf(&b, "%d beginbfchar\n", end-i)
		for _, id := range ids[i:end] {
			r := gidToRune[uint16(id)]
			fmt.Fprintf(&b, "<%04X> <%04X>\n", id, r)
		}
		b.WriteString("endbfchar\n")
	}
	b.WriteString("endcmap\nCMapName currentdict /CMap defineresource pop\nend\nend")
	return b.String()
}
