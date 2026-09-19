package project

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"dialogueforge/internal/circuit"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

func sample() Project {
	return Project{Request: circuit.Request{MaxWidth: 8, Target: "@a", Name: "合并剧情", Lines: []circuit.Line{
		{Text: "A：开始", SourceFile: "first.txt", SourceGroup: 1, DelayTenths: 3},
		{Text: "/scoreboard players add test points 1", SourceFile: "first.txt", SourceGroup: 1, DelayTenths: 7},
		{Text: "B：下一章", SourceFile: "second.txt", SourceGroup: 2, DelayTenths: 20, BreakBefore: true},
		{Text: "/give @a minecraft:diamond 1", SourceFile: "second.txt", SourceGroup: 2, DelayTenths: 0},
	}}}
}

func TestPartitionAndTiming(t *testing.T) {
	for _, mode := range []string{"none", "count", "files", "manual"} {
		p := sample()
		p.SplitMode, p.SplitEvery = mode, 2
		bundle, err := Build(p)
		if err != nil {
			t.Fatal(mode, err)
		}
		want := 2
		if mode == "none" {
			want = 1
		}
		if len(bundle.Parts) != want {
			t.Fatal(mode, bundle.Parts)
		}
		var got []string
		for i, l := range bundle.Layouts {
			if l.Events[0].AtTenths != 0 || l.Cells[0].Kind != "button" {
				t.Fatal("not independently playable")
			}
			for _, e := range l.Events {
				got = append(got, e.Text)
			}
			if mode != "none" && l.DurationTenths != p.Lines[bundle.Parts[i].Start].DelayTenths {
				t.Fatal("cross-part interval leaked")
			}
		}
		if len(got) != len(p.Lines) {
			t.Fatal("entries lost")
		}
		for i, s := range got {
			if s != p.Lines[i].Text {
				t.Fatal("order changed")
			}
		}
		preview, err := bundle.Preview(want - 1)
		if err != nil || preview.TotalLines != 4 || preview.PartIndex != want-1 {
			t.Fatal(preview, err)
		}
	}
}

func TestArchiveContentsAndCommands(t *testing.T) {
	p := sample()
	p.SplitMode, p.SplitEvery = "count", 2
	b, err := Build(p)
	if err != nil {
		t.Fatal(err)
	}
	data, name, err := b.Export(p, nil)
	if err != nil || name != "合并剧情.zip" {
		t.Fatal(name, err)
	}
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	if len(z.File) != 4 {
		t.Fatal("missing structures, manifest or project")
	}
	for i, f := range z.File {
		r, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		content, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			t.Fatal(err)
		}
		if i < 2 {
			gz, err := gzip.NewReader(bytes.NewReader(content))
			if err != nil {
				t.Fatal(err)
			}
			nbt, err := io.ReadAll(gz)
			gz.Close()
			if err != nil {
				t.Fatal(err)
			}
			command, _ := p.Lines[i*2+1].Command("@a")
			if !bytes.Contains(nbt, []byte(command)) || bytes.Contains(nbt, []byte("tellraw @a {\"text\":\"/")) {
				t.Fatal("raw command lost or wrapped")
			}
		} else if f.Name == "project.dialogue.json" {
			var restored Project
			if err := json.Unmarshal(content, &restored); err != nil {
				t.Fatal(err)
			}
			if restored.Lines[2].SourceFile != "second.txt" || !restored.Lines[2].BreakBefore {
				t.Fatal("project metadata lost")
			}
		}
	}
	index := 1
	single, name, err := b.Export(p, &index)
	if err != nil || !strings.HasSuffix(name, "_part_002.schem") || !bytes.HasPrefix(single, []byte{0x1f, 0x8b}) {
		t.Fatal(name, err)
	}
	index = 2
	if _, _, err = b.Export(p, &index); err == nil {
		t.Fatal("invalid part accepted")
	}
}

func TestLimitsAndLegacy(t *testing.T) {
	var legacy Project
	if err := json.Unmarshal([]byte(`{"name":"old","max_width":8,"target":"@a","lines":[{"text":"A","delay_tenths":0}]}`), &legacy); err != nil {
		t.Fatal(err)
	}
	if b, err := Build(legacy); err != nil || b.Layouts[0].LaneGap != 3 {
		t.Fatal(b, err)
	}
	p := sample()
	p.Lines = nil
	for i := 0; i < 2001; i++ {
		p.Lines = append(p.Lines, circuit.Line{Text: "A", DelayTenths: 1})
	}
	if _, err := Build(p); err == nil {
		t.Fatal("oversized single structure accepted")
	}
	p.SplitMode, p.SplitEvery = "count", 1000
	if b, err := Build(p); err != nil || len(b.Parts) != 3 {
		t.Fatal(b, err)
	}
	for _, n := range []int{0, -1, 2001} {
		p.SplitEvery = n
		if _, err := Build(p); err == nil {
			t.Fatal("invalid split count", n)
		}
	}
	p.SplitEvery = 1
	if _, err := Build(p); err == nil {
		t.Fatal("part limit missing")
	}
	p = sample()
	p.SplitMode = "files"
	p.Lines[2].SourceFile = "first.txt"
	parts, err := p.Partitions()
	if err != nil || len(parts) != 3 {
		t.Fatal("duplicate filenames must preserve distinct imports", parts, err)
	}
	for _, name := range []string{"../escape", "CON.txt", "", "a/b", "章节😀"} {
		s := SafeName(name)
		if strings.ContainsAny(s, "/\\") || s == "" || s == "CON.txt" {
			t.Fatal(s)
		}
	}
}
