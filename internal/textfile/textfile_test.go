package textfile

import "testing"

func TestLineBoundaries(t *testing.T) {
	p, e := Parse("\ufeffA：第一行\r\nB：第二行\r\n\r\nA：第三行\rA：第四行\n", true, 20)
	if e != nil {
		t.Fatal(e)
	}
	if len(p.Lines) != 4 || p.BlankLines != 1 {
		t.Fatal(p)
	}
	if p.Lines[2].SourceLine != 4 || p.Lines[3].Text != "A：第四行" {
		t.Fatal(p)
	}
	p, e = Parse("  A： 保留空格  \n\nB：结束\n", false, 20)
	if e != nil {
		t.Fatal(e)
	}
	if len(p.Lines) != 3 || p.Lines[0].Text != "  A： 保留空格  " || p.Lines[1].Text != "" {
		t.Fatal(p)
	}
	if _, e = Parse("\n\n", true, 20); e == nil {
		t.Fatal("empty")
	}
}

func TestDecode(t *testing.T) {
	for _, b := range [][]byte{{0xef, 0xbb, 0xbf, 'A'}, {0xff, 0xfe, 'A', 0}, {0xfe, 0xff, 0, 'A'}} {
		s, _, e := Decode(b)
		if e != nil || s != "A" {
			t.Fatal(s, e)
		}
	}
	if _, _, e := Decode([]byte{0xff}); e == nil {
		t.Fatal("invalid encoding")
	}
	if _, _, e := Decode([]byte{0xff, 0xfe, 0}); e == nil {
		t.Fatal("truncated UTF16")
	}
}
