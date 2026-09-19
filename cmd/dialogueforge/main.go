package main

import (
	"bytes"
	"dialogueforge/internal/app"
	"dialogueforge/internal/circuit"
	"dialogueforge/internal/schematic"
	"dialogueforge/internal/textfile"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if err := run(); err != nil {
		log.Print(err)
		if len(os.Args) == 1 {
			app.NotifyError(err.Error())
		}
		os.Exit(1)
	}
}
func run() error {
	in := flag.String("input", "", "输入 TXT（命令行模式；GUI 支持更多编码）")
	out := flag.String("output", "dialogue.schem", "输出 .schem")
	width := flag.Int("width", 64, "最大宽度，包含转弯，8～256 格")
	interval := flag.Float64("interval", 2, "默认每句后间隔（秒，0.1 的倍数）")
	target := flag.String("target", "@a", "tellraw 目标")
	project := flag.String("project", "", "读取界面保存的 .dialogue.json 项目")
	preview := flag.String("preview-json", "", "同时输出线路分析 JSON")
	keepBlank := flag.Bool("keep-blank", false, "把空行也作为一条消息")
	addr := flag.String("listen", "127.0.0.1:0", "本机界面监听地址")
	noBrowser := flag.Bool("no-browser", false, "启动界面但不自动打开浏览器")
	version := flag.Bool("version", false, "显示版本")
	flag.Parse()
	if *version {
		fmt.Println(app.Version)
		return nil
	}
	if *in == "" && *project == "" {
		return app.Serve(*addr, *noBrowser)
	}
	var r circuit.Request
	if *project != "" {
		b, err := os.ReadFile(*project)
		if err != nil {
			return err
		}
		if err = json.Unmarshal(b, &r); err != nil {
			return err
		}
	} else {
		b, err := os.ReadFile(*in)
		if err != nil {
			return err
		}
		if len(b) > 4<<20 {
			return fmt.Errorf("TXT 文件超过 4 MB")
		}
		s, _, err := textfile.Decode(b)
		if err != nil {
			return err
		}
		d, err := circuit.SecondsToTenths(*interval)
		if err != nil {
			return err
		}
		p, err := textfile.Parse(s, !*keepBlank, d)
		if err != nil {
			return err
		}
		r = circuit.Request{Lines: p.Lines, MaxWidth: *width, Target: *target, Name: strings.TrimSuffix(filepath.Base(*in), filepath.Ext(*in))}
	}
	l, err := circuit.Build(r)
	if err != nil {
		return err
	}
	// Encode before opening the destination so validation cannot truncate it.
	var buf bytes.Buffer
	if err = schematic.Write(&buf, l); err != nil {
		return err
	}
	if err = os.WriteFile(*out, buf.Bytes(), 0644); err != nil {
		return err
	}
	if *preview != "" {
		b, e := json.MarshalIndent(l, "", "  ")
		if e != nil {
			return e
		}
		if e = os.WriteFile(*preview, b, 0644); e != nil {
			return e
		}
	}
	fmt.Printf("已生成 %s：%d 条对白，%d 个中继器，%d × %d × %d 格，首句至末句 %.1f 秒\n", *out, len(l.Events), l.Repeaters, l.Width, l.Height, l.Length, float64(l.DurationTenths)/10)
	return nil
}
