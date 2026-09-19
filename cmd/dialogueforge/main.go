package main

import (
	"dialogueforge/internal/app"
	"dialogueforge/internal/circuit"
	"dialogueforge/internal/project"
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

type inputs []string

func (s *inputs) String() string         { return strings.Join(*s, ", ") }
func (s *inputs) Set(value string) error { *s = append(*s, value); return nil }
func run() error {
	var in inputs
	flag.Var(&in, "input", "输入 TXT；可多次指定，按参数顺序合并")
	out := flag.String("output", "", "输出 .schem 或多结构 .zip；默认按项目名命名")
	width := flag.Int("width", 64, "最大宽度，包含转弯，8～256 格")
	gap := flag.Int("gap", 3, "两条主线之间的空格数，至少为 1")
	splitMode := flag.String("split-mode", "none", "none 合并 / count 按条数 / files 按文件 / manual 按项目分界")
	splitEvery := flag.Int("split-every", 100, "按条数分块时，每块条数（1～2000）")
	interval := flag.Float64("interval", 2, "默认每句后间隔（秒，0.1 的倍数）")
	target := flag.String("target", "@a", "tellraw 目标")
	projectPath := flag.String("project", "", "读取界面保存的 .dialogue.json 项目（使用项目内设置）")
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
	if len(in) == 0 && *projectPath == "" {
		return app.Serve(*addr, *noBrowser)
	}
	if len(in) > 0 && *projectPath != "" {
		return fmt.Errorf("--input 与 --project 不能同时指定")
	}
	var p project.Project
	if *projectPath != "" {
		b, err := os.ReadFile(*projectPath)
		if err != nil {
			return err
		}
		if len(b) > 8<<20 {
			return fmt.Errorf("项目文件超过 8 MB")
		}
		if err = json.Unmarshal(b, &p); err != nil {
			return err
		}
	} else {
		d, err := circuit.SecondsToTenths(*interval)
		if err != nil {
			return err
		}
		p = project.Project{Request: circuit.Request{MaxWidth: *width, LaneGap: gap, Target: *target, Name: strings.TrimSuffix(filepath.Base(in[0]), filepath.Ext(in[0]))}, SchemaVersion: 2, SplitMode: *splitMode, SplitEvery: *splitEvery}
		if len(in) > 1 {
			p.Name += "_merged"
		}
		for group, filename := range in {
			b, err := os.ReadFile(filename)
			if err != nil {
				return err
			}
			if len(b) > 4<<20 {
				return fmt.Errorf("%s 超过 4 MB", filename)
			}
			s, _, err := textfile.Decode(b)
			if err != nil {
				return fmt.Errorf("%s：%w", filename, err)
			}
			parsed, err := textfile.Parse(s, !*keepBlank, d)
			if err != nil {
				return fmt.Errorf("%s：%w", filename, err)
			}
			for _, line := range parsed.Lines {
				line.SourceFile, line.SourceGroup = filepath.Base(filename), group+1
				p.Lines = append(p.Lines, line)
			}
			if len(p.Lines) > circuit.MaxProjectLines {
				return fmt.Errorf("一个项目最多 %d 条", circuit.MaxProjectLines)
			}
		}
	}
	bundle, err := project.Build(p)
	if err != nil {
		return err
	}
	// Encode before opening the destination so validation cannot truncate it.
	data, name, err := bundle.Export(p, nil)
	if err != nil {
		return err
	}
	if *out == "" {
		*out = name
	}
	wantExt := filepath.Ext(name)
	if strings.ToLower(filepath.Ext(*out)) != wantExt {
		return fmt.Errorf("本次导出 %d 个结构，输出扩展名应为 %s", len(bundle.Parts), wantExt)
	}
	if err = os.WriteFile(*out, data, 0644); err != nil {
		return err
	}
	if *preview != "" {
		layout, _ := bundle.Preview(0)
		b, e := json.MarshalIndent(layout, "", "  ")
		if e != nil {
			return e
		}
		if e = os.WriteFile(*preview, b, 0644); e != nil {
			return e
		}
	}
	fmt.Printf("已生成 %s：%d 条对白或指令，%d 个独立结构\n", *out, len(p.Lines), len(bundle.Parts))
	return nil
}
