package textfile

import (
	"bytes"
	"dialogueforge/internal/circuit"
	"encoding/binary"
	"fmt"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

type Parsed struct {
	Lines        []circuit.Line `json:"lines"`
	BlankLines   int            `json:"blank_lines"`
	CommentLines int            `json:"comment_lines"`
	Encoding     string         `json:"encoding"`
}

// Decode supports UTF-8 and BOM-marked UTF-16. The browser additionally supports
// GB18030 via TextDecoder, then sends normalized UTF-8 to this same parser.
func Decode(b []byte) (string, string, error) {
	if bytes.HasPrefix(b, []byte{0xef, 0xbb, 0xbf}) {
		b = b[3:]
	}
	if bytes.HasPrefix(b, []byte{0xff, 0xfe}) || bytes.HasPrefix(b, []byte{0xfe, 0xff}) {
		var order binary.ByteOrder = binary.LittleEndian
		enc := "UTF-16 LE"
		if b[0] == 0xfe {
			order = binary.BigEndian
			enc = "UTF-16 BE"
		}
		b = b[2:]
		if len(b)%2 != 0 {
			return "", "", fmt.Errorf("UTF-16 文件长度不完整")
		}
		units := make([]uint16, len(b)/2)
		for i := range units {
			units[i] = order.Uint16(b[i*2:])
		}
		return string(utf16.Decode(units)), enc, nil
	}
	if !utf8.Valid(b) {
		return "", "", fmt.Errorf("文件不是 UTF-8 / UTF-16；请在界面选择 GB18030 编码，或把文件另存为 UTF-8")
	}
	return string(b), "UTF-8", nil
}

func Parse(s string, skipBlank bool, delay int) (Parsed, error) {
	p := Parsed{Lines: []circuit.Line{}}
	s = strings.TrimPrefix(s, "\ufeff")
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
	// A terminator at EOF ends the final line; it does not create another line.
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return p, fmt.Errorf("TXT 文件没有文本")
	}
	for i, t := range strings.Split(s, "\n") {
		// Only a # in the first column marks a comment. Preserve source line
		// numbers after dropping comments so the editor can locate the original.
		if strings.HasPrefix(t, "#") {
			p.CommentLines++
			continue
		}
		if strings.ContainsRune(t, 0) {
			return p, fmt.Errorf("第 %d 行含有 NUL 字符，请检查文件编码", i+1)
		}
		if strings.TrimSpace(t) == "" {
			p.BlankLines++
			if skipBlank {
				continue
			}
		}
		line := circuit.Line{Text: t, SourceLine: i + 1, DelayTenths: delay}
		line.Kind = line.Type()
		p.Lines = append(p.Lines, line)
	}
	if len(p.Lines) == 0 {
		// A comments-only file can be part of a batch. The project builder
		// rejects a batch that contains no playable entries at all.
		return p, nil
	}
	if len(p.Lines) > circuit.MaxProjectLines {
		return p, fmt.Errorf("一个项目最多支持 %d 条（不含已忽略的行）", circuit.MaxProjectLines)
	}
	return p, nil
}
