// Package circuit turns dialogue lines into a physical, vanilla redstone circuit.
package circuit

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

const (
	DataVersion     = 3955 // Minecraft Java 1.21.1; confirmed against the server version.json.
	MaxLines        = 2000
	MaxProjectLines = 10000
	MaxPathCells    = 100000
	MaxVolume       = 2000000
	DefaultLaneGap  = 3
)

type Line struct {
	Text        string `json:"text"`
	SourceLine  int    `json:"source_line"`
	SourceFile  string `json:"source_file,omitempty"`
	SourceGroup int    `json:"source_group,omitempty"`
	Kind        string `json:"kind,omitempty"`
	BreakBefore bool   `json:"break_before,omitempty"`
	// DelayTenths is the interval AFTER this line. The final line has no successor.
	DelayTenths int `json:"delay_tenths"`
}

type Request struct {
	Lines    []Line `json:"lines"`
	MaxWidth int    `json:"max_width"`
	Target   string `json:"target"`
	Name     string `json:"name"`
	// A pointer distinguishes an omitted legacy setting from the invalid value 0.
	LaneGap *int `json:"lane_gap,omitempty"`
}

func (r Request) Gap() int {
	if r.LaneGap == nil {
		return DefaultLaneGap
	}
	return *r.LaneGap
}

func (l Line) Type() string {
	if l.Kind != "" {
		return l.Kind
	}
	if strings.HasPrefix(l.Text, "/") {
		return "command"
	}
	return "dialogue"
}

func (l Line) Command(target string) (string, error) {
	switch l.Type() {
	case "command":
		command := strings.TrimSpace(strings.TrimPrefix(l.Text, "/"))
		if command == "" {
			return "", fmt.Errorf("指令不能为空，请补全 / 后的内容")
		}
		return command, nil
	case "dialogue":
		payload, _ := json.Marshal(map[string]string{"text": l.Text})
		return "tellraw " + target + " " + string(payload), nil
	default:
		return "", fmt.Errorf("类型应为 dialogue 或 command")
	}
}

type Pos struct {
	X int `json:"x"`
	Y int `json:"y"`
	Z int `json:"z"`
}
type Cell struct {
	Pos
	Kind   string `json:"kind"`
	Facing string `json:"facing,omitempty"`
	Delay  int    `json:"delay,omitempty"`
	Line   int    `json:"line,omitempty"`
	State  string `json:"-"`
}
type Event struct {
	Pos
	Line           int    `json:"line"`
	SourceLine     int    `json:"source_line"`
	Text           string `json:"text"`
	AtTenths       int    `json:"at_tenths"`
	RepeatersAfter int    `json:"repeaters_after"`
	Command        string `json:"command"`
	Kind           string `json:"kind"`
	SourceFile     string `json:"source_file,omitempty"`
}
type Layout struct {
	Width          int     `json:"width"`
	Height         int     `json:"height"`
	Length         int     `json:"length"`
	Repeaters      int     `json:"repeaters"`
	DurationTenths int     `json:"duration_tenths"`
	Rows           int     `json:"rows"`
	Cells          []Cell  `json:"cells"`
	Events         []Event `json:"events"`
	Name           string  `json:"name"`
	LaneGap        int     `json:"lane_gap"`
}

func SecondsToTenths(s float64) (int, error) {
	if math.IsNaN(s) || math.IsInf(s, 0) || s < .1 || s > 600 {
		return 0, fmt.Errorf("间隔应为 0.1～600 秒")
	}
	n := math.Round(s * 10)
	if math.Abs(s*10-n) > .00001 {
		return 0, fmt.Errorf("中继器的时间精度为 0.1 秒，请最多填写一位小数")
	}
	return int(n), nil
}

func (r Request) Validate() error {
	if len(r.Lines) == 0 {
		return fmt.Errorf("请先导入至少一行文本")
	}
	if len(r.Lines) > MaxLines {
		return fmt.Errorf("每个结构最多 %d 条，请使用分块导出", MaxLines)
	}
	if r.MaxWidth < 8 || r.MaxWidth > 256 {
		return fmt.Errorf("最大宽度应为 8～256 格（包含按钮及转弯）")
	}
	if r.Gap() < 1 || r.Gap() > 32758 {
		return fmt.Errorf("两条主线之间的空格数应为 1～32758，不能为 0；最终结构仍受体积限制")
	}
	if r.Target == "" || len(r.Target) > 1024 || strings.ContainsAny(r.Target, "\r\n\x00") {
		return fmt.Errorf("播报对象无效")
	}
	if !(strings.HasPrefix(r.Target, "@a") || strings.HasPrefix(r.Target, "@p") || strings.HasPrefix(r.Target, "@r") || strings.HasPrefix(r.Target, "@s")) {
		return fmt.Errorf("播报对象请使用 @a、@p、@r 或 @s，可附加选择器条件")
	}
	if len(r.Target) > 2 && !(r.Target[2] == '[' && r.Target[len(r.Target)-1] == ']') {
		return fmt.Errorf("播报对象的选择器条件应放在方括号内")
	}
	cells := len(r.Lines)
	for i, l := range r.Lines {
		if !utf8.ValidString(l.Text) || strings.ContainsAny(l.Text, "\r\n\x00") {
			return fmt.Errorf("第 %d 条包含无效字符或换行", i+1)
		}
		if len(l.Text) > 16000 {
			return fmt.Errorf("第 %d 条文本过长，请拆分成多行", i+1)
		}
		command, err := l.Command(r.Target)
		if err != nil {
			return fmt.Errorf("第 %d 条：%w", i+1, err)
		}
		if len(command) > 30000 {
			return fmt.Errorf("第 %d 条转换后的指令过长，请拆分", i+1)
		}
		if i < len(r.Lines)-1 {
			if l.DelayTenths < 1 || l.DelayTenths > 6000 {
				return fmt.Errorf("第 %d 条的间隔应为 0.1～600 秒", i+1)
			}
			cells += (l.DelayTenths + 3) / 4
		}
	}
	if cells > MaxPathCells {
		return fmt.Errorf("线路过大，请减少文本数量或间隔（最多 %d 个线路元件）", MaxPathCells)
	}
	return nil
}

// cursor produces a non-self-touching serpentine path. Both corner cells are
// always dust, so a bend NEVER adds hidden repeater delay. A gap of at least
// one block keeps neighboring lanes from connecting or locking repeaters.
type cursor struct{ x, z, row, phase, width, spacing int }

func newCursor(w, spacing int) *cursor { return &cursor{x: 1, width: w, spacing: spacing} }
func (c *cursor) pos() Pos             { return Pos{c.x, 1, c.z} }
func (c *cursor) next() Pos {
	if c.phase > 0 {
		c.z++
		c.phase++
		if c.phase == c.spacing+1 {
			c.phase = 0
			c.row++
		}
	} else {
		end := c.width - 2
		if c.row%2 == 1 {
			end = 1
		}
		if c.x == end {
			c.z++
			c.phase = 2
		} else if c.row%2 == 0 {
			c.x++
		} else {
			c.x--
		}
	}
	return c.pos()
}

func direction(a, b Pos) string {
	if b.X > a.X {
		return "east"
	}
	if b.X < a.X {
		return "west"
	}
	if b.Z > a.Z {
		return "south"
	}
	return "north"
}
func opposite(s string) string {
	return map[string]string{"east": "west", "west": "east", "north": "south", "south": "north"}[s]
}
func wireState(in, out string) string {
	parts := []string{}
	for _, d := range []string{"east", "north", "south", "west"} {
		v := "none"
		if d == opposite(in) || d == out {
			v = "side"
		}
		parts = append(parts, d+"="+v)
	}
	return "minecraft:redstone_wire[" + strings.Join(parts, ",") + ",power=0]"
}

func Build(r Request) (*Layout, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	l := &Layout{Height: 3, Name: r.Name, LaneGap: r.Gap(), Cells: []Cell{{Pos: Pos{0, 1, 0}, Kind: "button", State: "minecraft:stone_button[face=floor,facing=east,powered=false]"}}}
	if l.Name == "" {
		l.Name = "dialogue"
	}
	c := newCursor(r.MaxWidth, r.Gap()+1)
	prev := Pos{0, 1, 0}
	current := c.pos()
	next := c.next()
	lineIndex, pendingDelay, elapsed, maxX, maxZ := 0, 0, 0, 0, 0
	for {
		in, out := direction(prev, current), direction(current, next)
		cell := Cell{Pos: current}
		if pendingDelay == 0 {
			line := r.Lines[lineIndex]
			command, _ := line.Command(r.Target) // Already validated above.
			count := 0
			if lineIndex < len(r.Lines)-1 {
				count = (line.DelayTenths + 3) / 4
			}
			l.Events = append(l.Events, Event{Pos: Pos{current.X, 0, current.Z}, Line: lineIndex + 1, SourceLine: line.SourceLine, SourceFile: line.SourceFile, Kind: line.Type(), Text: line.Text, AtTenths: elapsed, RepeatersAfter: count, Command: command})
			cell.Kind = line.Type()
			cell.Line = lineIndex + 1
			cell.State = wireState(in, out)
			lineIndex++
			if lineIndex < len(r.Lines) {
				pendingDelay = line.DelayTenths
			}
		} else if in == out {
			d := pendingDelay
			if d > 4 {
				d = 4
			}
			cell.Kind = "repeater"
			// Facing in the preview is signal travel. Minecraft's repeater block
			// state points toward its input, so the serialized value is opposite.
			cell.Facing = out
			cell.Delay = d
			cell.State = fmt.Sprintf("minecraft:repeater[delay=%d,facing=%s,locked=false,powered=false]", d, opposite(out))
			pendingDelay -= d
			elapsed += d
			l.Repeaters++
		} else {
			cell.Kind = "wire"
			cell.State = wireState(in, out)
		}
		l.Cells = append(l.Cells, cell)
		if current.X > maxX {
			maxX = current.X
		}
		if current.Z > maxZ {
			maxZ = current.Z
		}
		if lineIndex == len(r.Lines) {
			break
		}
		if len(l.Cells) > MaxPathCells*2 || maxZ > 32760 {
			return nil, fmt.Errorf("结构过大，请缩短间隔或增加宽度")
		}
		prev = current
		current = next
		next = c.next()
	}
	l.Width = maxX + 2
	l.Length = maxZ + 2
	l.DurationTenths = elapsed
	l.Rows = maxZ/(r.Gap()+1) + 1
	if l.Width*l.Height*l.Length > MaxVolume {
		return nil, fmt.Errorf("结构体积超过 %d 格，请减少文本或间隔", MaxVolume)
	}
	return l, nil
}

// Blocks includes an insulating glass floor and explicit air clearance. Command
// blocks sit BELOW their own dust tap. Glass prevents a powered support block
// from triggering an adjacent command early; commands themselves never relay
// power to the next command. Playback is driven solely by the continuous bus.
func (l *Layout) Blocks() []string {
	b := make([]string, l.Width*l.Height*l.Length)
	for i := range b {
		b[i] = "minecraft:air"
	}
	for z := 0; z < l.Length; z++ {
		for x := 0; x < l.Width; x++ {
			b[x+z*l.Width] = "minecraft:glass"
		}
	}
	put := func(p Pos, s string) { b[p.X+p.Z*l.Width+p.Y*l.Width*l.Length] = s }
	put(Pos{0, 0, 0}, "minecraft:gold_block")
	for _, c := range l.Cells {
		put(c.Pos, c.State)
	}
	for _, e := range l.Events {
		put(e.Pos, "minecraft:command_block[conditional=false,facing=up]")
	}
	return b
}
