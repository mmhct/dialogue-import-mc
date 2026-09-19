// Package project preserves the editable timeline and partitions it into
// independently playable structures. Splitting never silently drops entries.
package project

import (
	"archive/zip"
	"bytes"
	"dialogueforge/internal/circuit"
	"dialogueforge/internal/schematic"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
)

const MaxParts = 256
const MaxTotalVolume = 10000000

type Project struct {
	circuit.Request
	SchemaVersion int    `json:"schema_version,omitempty"`
	SplitMode     string `json:"split_mode,omitempty"`
	SplitEvery    int    `json:"split_every,omitempty"`
}

type Part struct {
	Index          int    `json:"index"`
	Name           string `json:"name"`
	Start          int    `json:"start"` // Inclusive, zero-based index in the project.
	End            int    `json:"end"`   // Exclusive.
	Width          int    `json:"width"`
	Length         int    `json:"length"`
	Height         int    `json:"height"`
	DurationTenths int    `json:"duration_tenths"`
	Repeaters      int    `json:"repeaters"`
}

type Bundle struct {
	Layouts []*circuit.Layout
	Parts   []Part
}

type Preview struct {
	*circuit.Layout        // Keep the v1 single-layout JSON shape compatible.
	Parts           []Part `json:"parts"`
	PartIndex       int    `json:"part_index"`
	TotalLines      int    `json:"total_lines"`
}

func (p Project) Partitions() ([]Part, error) {
	if p.SchemaVersion < 0 || p.SchemaVersion > 2 {
		return nil, fmt.Errorf("此项目版本不受支持，请升级软件")
	}
	if len(p.Lines) == 0 || len(p.Lines) > circuit.MaxProjectLines {
		return nil, fmt.Errorf("一个项目需要 1～%d 条对白或指令", circuit.MaxProjectLines)
	}
	mode := p.SplitMode
	if mode == "" {
		mode = "none"
	}
	if mode != "none" && mode != "count" && mode != "manual" && mode != "files" {
		return nil, fmt.Errorf("未知分块方式")
	}
	if mode == "count" && (p.SplitEvery < 1 || p.SplitEvery > circuit.MaxLines) {
		return nil, fmt.Errorf("每块条数应为 1～%d", circuit.MaxLines)
	}
	parts := []Part{{Start: 0}}
	for i := 1; i < len(p.Lines); i++ {
		line, prev := p.Lines[i], p.Lines[i-1]
		cut := mode == "count" && i%p.SplitEvery == 0 || mode == "manual" && line.BreakBefore || mode == "files" && (line.SourceGroup != prev.SourceGroup || line.SourceFile != prev.SourceFile)
		if cut {
			parts[len(parts)-1].End = i
			parts = append(parts, Part{Start: i})
			if len(parts) > MaxParts {
				return nil, fmt.Errorf("最多导出 %d 个结构，请增加每块条数或减少分界", MaxParts)
			}
		}
	}
	parts[len(parts)-1].End = len(p.Lines)
	base := SafeName(p.Name)
	for i := range parts {
		parts[i].Index, parts[i].Name = i, base
		if len(parts) > 1 {
			parts[i].Name = fmt.Sprintf("%s_part_%03d", base, i+1)
		}
	}
	return parts, nil
}

func Build(p Project) (*Bundle, error) {
	parts, err := p.Partitions()
	if err != nil {
		return nil, err
	}
	b := &Bundle{Parts: parts}
	volume := 0
	for i := range parts {
		part := &b.Parts[i]
		r := p.Request
		r.Lines, r.Name = p.Lines[part.Start:part.End], part.Name
		layout, err := circuit.Build(r)
		if err != nil {
			return nil, fmt.Errorf("结构 %d（第 %d～%d 条）：%w", i+1, part.Start+1, part.End, err)
		}
		volume += layout.Width * layout.Height * layout.Length
		if volume > MaxTotalVolume {
			return nil, fmt.Errorf("本次导出总体积超过 %d 格，请减少条数或间隔", MaxTotalVolume)
		}
		part.Width, part.Length, part.Height = layout.Width, layout.Length, layout.Height
		part.DurationTenths, part.Repeaters = layout.DurationTenths, layout.Repeaters
		b.Layouts = append(b.Layouts, layout)
	}
	return b, nil
}

func (b *Bundle) Preview(index int) (*Preview, error) {
	if index < 0 || index >= len(b.Layouts) {
		return nil, fmt.Errorf("结构编号不存在")
	}
	return &Preview{Layout: b.Layouts[index], Parts: b.Parts, PartIndex: index, TotalLines: b.Parts[len(b.Parts)-1].End}, nil
}

// Export completes encoding in memory before the caller opens a destination.
// Each part starts at time zero with its own button; there is no cross-part wire.
func (b *Bundle) Export(p Project, onlyPart *int) ([]byte, string, error) {
	if onlyPart != nil || len(b.Layouts) == 1 {
		i := 0
		if onlyPart != nil {
			i = *onlyPart
		}
		if i < 0 || i >= len(b.Layouts) {
			return nil, "", fmt.Errorf("结构编号不存在")
		}
		var buf bytes.Buffer
		if err := schematic.Write(&buf, b.Layouts[i]); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), b.Parts[i].Name + ".schem", nil
	}
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	for i, l := range b.Layouts {
		w, err := z.Create(b.Parts[i].Name + ".schem")
		if err != nil {
			return nil, "", err
		}
		if err = schematic.Write(w, l); err != nil {
			return nil, "", err
		}
	}
	manifest := struct {
		Parts []Part `json:"parts"`
		Note  string `json:"note"`
	}{b.Parts, "每个结构独立播放，各有启动按钮；每块最后一条的间隔不生效，块之间不会自动连接。start 从 0 开始，end 为不包含的结束位置。"}
	for _, item := range []struct {
		name  string
		value any
	}{{"manifest.json", manifest}, {"project.dialogue.json", p}} {
		w, err := z.Create(item.name)
		if err != nil {
			return nil, "", err
		}
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		if err = encoder.Encode(item.value); err != nil {
			return nil, "", err
		}
	}
	if err := z.Close(); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), SafeName(p.Name) + ".zip", nil
}

func SafeName(name string) string {
	var result []rune
	for _, r := range name {
		if unicode.IsControl(r) || strings.ContainsRune(`<>:"/\|?*`, r) {
			r = '_'
		}
		result = append(result, r)
		if len(result) == 80 {
			break
		}
	}
	s := strings.TrimRight(string(result), ". ")
	first := strings.ToUpper(strings.SplitN(s, ".", 2)[0])
	reserved := first == "CON" || first == "PRN" || first == "AUX" || first == "NUL" || len(first) == 4 && (strings.HasPrefix(first, "COM") || strings.HasPrefix(first, "LPT")) && first[3] >= '1' && first[3] <= '9'
	if s == "" || reserved {
		s = "dialogue_" + s
	}
	return s
}
