package schematic

import (
	"encoding/binary"
	"fmt"
	"io"
	"unicode/utf16"
)

// WorldEdit 7.3.8 reads NBT strings with Java DataInput.readUTF. Encode UTF-16
// surrogate pairs as modified UTF-8; ordinary four-byte UTF-8 fails to load.
type nbtWriter struct {
	w   io.Writer
	err error
}

func (n *nbtWriter) raw(p []byte) {
	if n.err == nil {
		_, n.err = n.w.Write(p)
	}
}
func (n *nbtWriter) u8(v byte) { n.raw([]byte{v}) }
func (n *nbtWriter) i16(v int) {
	var b [2]byte
	binary.BigEndian.PutUint16(b[:], uint16(v))
	n.raw(b[:])
}
func (n *nbtWriter) i32(v int) {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], uint32(v))
	n.raw(b[:])
}
func (n *nbtWriter) str(s string) {
	b := make([]byte, 0, len(s))
	for _, v := range utf16.Encode([]rune(s)) {
		switch {
		case v >= 1 && v <= 0x7f:
			b = append(b, byte(v))
		case v <= 0x7ff:
			b = append(b, byte(0xc0|v>>6), byte(0x80|v&0x3f))
		default:
			b = append(b, byte(0xe0|v>>12), byte(0x80|(v>>6)&0x3f), byte(0x80|v&0x3f))
		}
	}
	if len(b) > 65535 {
		n.err = fmt.Errorf("NBT 字符串超过 65535 字节")
		return
	}
	n.i16(len(b))
	n.raw(b)
}
func (n *nbtWriter) tag(t byte, name string)     { n.u8(t); n.str(name) }
func (n *nbtWriter) integer(name string, v int)  { n.tag(3, name); n.i32(v) }
func (n *nbtWriter) short(name string, v int)    { n.tag(2, name); n.i16(v) }
func (n *nbtWriter) byteTag(name string, v byte) { n.tag(1, name); n.u8(v) }
func (n *nbtWriter) stringTag(name, v string)    { n.tag(8, name); n.str(v) }
func (n *nbtWriter) ints(name string, v ...int) {
	n.tag(11, name)
	n.i32(len(v))
	for _, x := range v {
		n.i32(x)
	}
}
func (n *nbtWriter) bytes(name string, b []byte) { n.tag(7, name); n.i32(len(b)); n.raw(b) }
