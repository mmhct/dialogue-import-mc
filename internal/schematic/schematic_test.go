package schematic

import (
	"bytes"
	"compress/gzip"
	"dialogueforge/internal/circuit"
	"io"
	"testing"
)

func TestModifiedUTF8(t *testing.T) {
	var b bytes.Buffer
	n := nbtWriter{w: &b}
	n.str("A\x00😀")
	want := []byte{0, 9, 65, 0xc0, 0x80, 0xed, 0xa0, 0xbd, 0xed, 0xb8, 0x80}
	if !bytes.Equal(b.Bytes(), want) {
		t.Fatalf("%x", b.Bytes())
	}
}
func TestSchematicEnvelope(t *testing.T) {
	l, e := circuit.Build(circuit.Request{MaxWidth: 8, Target: "@a", Lines: []circuit.Line{{Text: "A：你好", DelayTenths: 20}, {Text: "B：😀"}}})
	if e != nil {
		t.Fatal(e)
	}
	var a, b bytes.Buffer
	if e = Write(&a, l); e != nil {
		t.Fatal(e)
	}
	if e = Write(&b, l); e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Fatal("non-reproducible export")
	}
	g, e := gzip.NewReader(&a)
	if e != nil {
		t.Fatal(e)
	}
	raw, e := io.ReadAll(g)
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.HasPrefix(raw, []byte{10, 0, 0, 10, 0, 9, 'S', 'c', 'h', 'e', 'm', 'a', 't', 'i', 'c'}) {
		t.Fatal("not Sponge v3 root")
	}
	if !bytes.Contains(raw, []byte("minecraft:command_block")) {
		t.Fatal("block entities missing")
	}
}
