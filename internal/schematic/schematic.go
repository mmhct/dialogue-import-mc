// Package schematic writes Sponge Schematic v3 files for WorldEdit 7.3+.
package schematic

import (
	"compress/gzip"
	"dialogueforge/internal/circuit"
	"io"
	"sort"
)

func Write(w io.Writer, l *circuit.Layout) error {
	g := gzip.NewWriter(w)
	n := &nbtWriter{w: g}
	n.tag(10, "")
	n.tag(10, "Schematic")
	n.integer("Version", 3)
	n.integer("DataVersion", circuit.DataVersion)
	n.short("Width", l.Width)
	n.short("Height", l.Height)
	n.short("Length", l.Length)
	n.ints("Offset", 0, 0, 0)
	n.tag(10, "Metadata")
	n.stringTag("Name", l.Name)
	n.stringTag("Author", "DialogueForge")
	n.stringTag("Description", "TXT dialogue; physical repeaters; Minecraft Java 1.21.1")
	n.u8(0)
	blocks := l.Blocks()
	states := map[string]int{}
	for _, s := range blocks {
		states[s] = 0
	}
	palette := make([]string, 0, len(states))
	for s := range states {
		palette = append(palette, s)
	}
	sort.Strings(palette)
	n.tag(10, "Blocks")
	n.tag(10, "Palette")
	for i, s := range palette {
		states[s] = i
		n.integer(s, i)
	}
	n.u8(0)
	data := make([]byte, 0, len(blocks))
	for _, s := range blocks {
		v := states[s]
		for v >= 128 {
			data = append(data, byte(v&127)|128)
			v >>= 7
		}
		data = append(data, byte(v))
	}
	n.bytes("Data", data)
	n.tag(9, "BlockEntities")
	n.u8(10)
	n.i32(len(l.Events))
	for _, e := range l.Events {
		n.ints("Pos", e.X, e.Y, e.Z)
		n.stringTag("Id", "minecraft:command_block")
		n.tag(10, "Data")
		n.stringTag("Command", e.Command)
		n.byteTag("auto", 0)
		n.byteTag("powered", 0)
		n.byteTag("conditionMet", 0)
		n.byteTag("TrackOutput", 0)
		n.integer("SuccessCount", 0)
		n.byteTag("UpdateLastExecution", 1)
		n.u8(0)
		n.u8(0)
	}
	n.u8(0)
	n.u8(0)
	n.u8(0)
	if n.err != nil {
		g.Close()
		return n.err
	}
	return g.Close()
}
