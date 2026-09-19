package app

import (
	"context"
	"crypto/rand"
	"dialogueforge/internal/project"
	"dialogueforge/internal/textfile"
	"dialogueforge/web"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const Version = "1.1.0"

func sendJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)
	sendJSON(w, map[string]string{"error": err.Error()})
}
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 8<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return fmt.Errorf("输入格式无效：%v", err)
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("输入中有多余内容")
	}
	return nil
}

func Handler(prefix string, shutdown func()) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' blob: data:; connect-src 'self'; frame-ancestors 'none'")
		if !strings.HasPrefix(r.URL.Path, prefix) {
			http.NotFound(w, r)
			return
		}
		path := strings.TrimPrefix(r.URL.Path, prefix)
		if strings.HasPrefix(path, "api/") {
			if r.Method != "POST" {
				http.Error(w, "POST required", 405)
				return
			}
			if origin := r.Header.Get("Origin"); origin != "" && origin != "http://"+r.Host {
				http.Error(w, "Origin rejected", 403)
				return
			}
			switch path {
			case "api/parse":
				var in struct {
					Text        string `json:"text"`
					SkipBlank   bool   `json:"skip_blank"`
					DelayTenths int    `json:"delay_tenths"`
				}
				if err := decode(w, r, &in); err != nil {
					fail(w, err)
					return
				}
				p, err := textfile.Parse(in.Text, in.SkipBlank, in.DelayTenths)
				if err != nil {
					fail(w, err)
					return
				}
				sendJSON(w, p)
			case "api/preview", "api/export", "api/export-part":
				var in struct {
					project.Project
					PartIndex int `json:"part_index"`
				}
				if err := decode(w, r, &in); err != nil {
					fail(w, err)
					return
				}
				bundle, err := project.Build(in.Project)
				if err != nil {
					fail(w, err)
					return
				}
				if path == "api/preview" {
					preview, err := bundle.Preview(in.PartIndex)
					if err != nil {
						fail(w, err)
						return
					}
					sendJSON(w, preview)
					return
				}
				var onlyPart *int
				if path == "api/export-part" {
					onlyPart = &in.PartIndex
				}
				data, name, err := bundle.Export(in.Project, onlyPart)
				if err != nil {
					fail(w, err)
					return
				}
				w.Header().Set("Content-Type", "application/octet-stream")
				if strings.HasSuffix(name, ".zip") {
					w.Header().Set("Content-Type", "application/zip")
				}
				w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
				w.Write(data)
			case "api/shutdown":
				sendJSON(w, map[string]bool{"ok": true})
				if shutdown != nil {
					go func() { time.Sleep(250 * time.Millisecond); shutdown() }()
				}
			default:
				http.NotFound(w, r)
			}
			return
		}
		if r.Method != "GET" {
			http.Error(w, "GET required", 405)
			return
		}
		if path == "" {
			path = "index.html"
		}
		if path != "index.html" && path != "app.js" && path != "style.css" {
			http.NotFound(w, r)
			return
		}
		b, err := web.Files.ReadFile(path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		kind := map[string]string{"index.html": "text/html", "app.js": "text/javascript", "style.css": "text/css"}[path]
		w.Header().Set("Content-Type", kind+"; charset=utf-8")
		w.Write(b)
	})
}

func Serve(addr string, noBrowser bool) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	if host != "127.0.0.1" && host != "localhost" && host != "::1" {
		return fmt.Errorf("程序只支持绑定本机地址")
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	token := make([]byte, 16)
	if _, err = rand.Read(token); err != nil {
		ln.Close()
		return err
	}
	prefix := "/" + hex.EncodeToString(token) + "/"
	server := &http.Server{ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 60 * time.Second}
	server.Handler = Handler(prefix, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	})
	url := "http://" + ln.Addr().String() + prefix
	log.Printf("DialogueForge %s — %s", Version, url)
	if !noBrowser {
		go func() {
			time.Sleep(150 * time.Millisecond)
			if e := openBrowser(url); e != nil {
				NotifyError(fmt.Sprintf("浏览器未能自动打开，请复制此地址到浏览器：\n%s\n\n%v", url, e))
			}
		}()
	}
	err = server.Serve(ln)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		return openWindowsBrowser(url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Run()
}
