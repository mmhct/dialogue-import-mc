package circuit

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"
)

func requestFor(n, width, delay int) Request {
	r := Request{MaxWidth: width, Target: "@a"}
	for i := 0; i < n; i++ {
		r.Lines = append(r.Lines, Line{Text: fmt.Sprintf("A：第 %d 行", i+1), SourceLine: i + 1, DelayTenths: delay})
	}
	return r
}
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func TestPhysicalPathAndTiming(t *testing.T) {
	for _, width := range []int{8, 9, 16, 31, 64, 256} {
		for _, delay := range []int{1, 2, 3, 4, 5, 7, 10, 20, 55, 80} {
			t.Run(fmt.Sprintf("width%d_delay%d", width, delay), func(t *testing.T) {
				r := requestFor(37, width, delay)
				l, err := Build(r)
				if err != nil {
					t.Fatal(err)
				}
				if l.Width > width {
					t.Fatalf("width %d > %d", l.Width, width)
				}
				if len(l.Events) != 37 || l.Repeaters != 36*((delay+3)/4) || l.DurationTenths != 36*delay {
					t.Fatal("event/delay count mismatch")
				}
				visited := map[Pos]int{}
				ticks := 0
				event := 0
				wireRun := 0
				for i, c := range l.Cells {
					if _, ok := visited[c.Pos]; ok {
						t.Fatalf("overlap at %+v", c.Pos)
					}
					visited[c.Pos] = i
					if c.X < 0 || c.X >= l.Width || c.Z < 0 || c.Z >= l.Length {
						t.Fatalf("out of bounds %+v", c)
					}
					if i > 0 {
						p := l.Cells[i-1].Pos
						if abs(p.X-c.X)+abs(p.Z-c.Z) != 1 {
							t.Fatal("disconnected path")
						}
					}
					if c.Kind == "repeater" {
						if c.Delay < 1 || c.Delay > 4 {
							t.Fatal("invalid delay")
						}
						ticks += c.Delay
						wireRun = 0
						if i == 0 || i == len(l.Cells)-1 {
							t.Fatal("repeater at endpoint")
						}
						if direction(l.Cells[i-1].Pos, c.Pos) != c.Facing || direction(c.Pos, l.Cells[i+1].Pos) != c.Facing {
							t.Fatalf("repeater at corner: %+v", c)
						}
					} else {
						wireRun++
						if wireRun > 14 {
							t.Fatal("wire loses signal strength")
						}
					}
					if c.Kind == "dialogue" {
						if ticks != event*delay || l.Events[event].AtTenths != ticks {
							t.Fatalf("wrong physical delay at %d: %d", event, ticks)
						}
						event++
					}
				}
				// Any side contact with a non-neighbor in the path is an electrical shortcut.
				for p, i := range visited {
					for _, d := range []Pos{{X: 1}, {X: -1}, {Z: 1}, {Z: -1}} {
						if j, ok := visited[Pos{p.X + d.X, p.Y, p.Z + d.Z}]; ok && abs(i-j) > 1 {
							t.Fatalf("short circuit at %v and step %d", p, j)
						}
					}
				}
				b := l.Blocks()
				if len(b) != l.Width*l.Height*l.Length {
					t.Fatal("volume")
				}
				for _, e := range l.Events {
					idx := e.X + e.Z*l.Width
					if !strings.HasPrefix(b[idx], "minecraft:command_block") {
						t.Fatal("missing command")
					}
					if !strings.HasPrefix(b[idx+l.Width*l.Length], "minecraft:redstone_wire") {
						t.Fatal("missing tap")
					}
				}
			})
		}
	}
}

func TestPerLineIntervalsAndText(t *testing.T) {
	r := requestFor(5, 8, 20)
	r.Lines[0].Text = "A：他说\"走吧\"，路径 C:\\地图 😀"
	delays := []int{1, 39, 7, 55, 6000}
	for i := range r.Lines {
		r.Lines[i].DelayTenths = delays[i]
	}
	l, err := Build(r)
	if err != nil {
		t.Fatal(err)
	}
	want := []int{0, 1, 40, 47, 102}
	for i, e := range l.Events {
		if e.AtTenths != want[i] {
			t.Fatal(i, e.AtTenths)
		}
	}
	var v map[string]string
	if err = json.Unmarshal([]byte(strings.TrimPrefix(l.Events[0].Command, "tellraw @a ")), &v); err != nil {
		t.Fatal(err)
	}
	if v["text"] != r.Lines[0].Text {
		t.Fatal("text was changed")
	}
	if l.DurationTenths != 102 {
		t.Fatal("last interval must not add circuitry")
	}
}

func TestSingleLineAndValidation(t *testing.T) {
	r := requestFor(1, 8, 0)
	l, err := Build(r)
	if err != nil {
		t.Fatal(err)
	}
	if l.Repeaters != 0 || len(l.Cells) != 2 || l.DurationTenths != 0 {
		t.Fatal("single line")
	}
	cases := []Request{requestFor(0, 8, 20), requestFor(2, 7, 20), requestFor(2, 257, 20), requestFor(2, 64, 0), requestFor(2, 64, 6001), requestFor(2001, 64, 20), requestFor(2000, 8, 6000)}
	for _, r := range cases {
		if _, err := Build(r); err == nil {
			t.Fatal("expected invalid request")
		}
	}
	for _, s := range []float64{0, -1, .15, 600.1, math.NaN(), math.Inf(1)} {
		if _, err := SecondsToTenths(s); err == nil {
			t.Fatal(s)
		}
	}
	for _, s := range []float64{.1, .3, 2, 3.9, 600} {
		if _, err := SecondsToTenths(s); err != nil {
			t.Fatal(err)
		}
	}
}
