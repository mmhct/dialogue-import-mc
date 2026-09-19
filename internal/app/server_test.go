package app

import (
	"archive/zip"
	"bytes"
	"dialogueforge/internal/circuit"
	"dialogueforge/internal/project"
	"encoding/json"
	"mime"
	"net/http/httptest"
	"testing"
)

func TestExportAndPreview(t *testing.T) {
	h := Handler("/test/", nil)
	in := circuit.Request{MaxWidth: 8, Target: "@a", Lines: []circuit.Line{{Text: "A：你好", DelayTenths: 30}, {Text: "B：好"}}}
	data, _ := json.Marshal(in)
	for _, path := range []string{"preview", "export"} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "http://localhost/test/api/"+path, bytes.NewReader(data))
		h.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		if path == "export" && !bytes.HasPrefix(w.Body.Bytes(), []byte{0x1f, 0x8b}) {
			t.Fatal("not gzip")
		}
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "http://localhost/test/api/export", bytes.NewReader(data))
	r.Header.Set("Origin", "https://example.com")
	h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("cross-origin accepted")
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost/other/", nil))
	if w.Code != 404 {
		t.Fatal("prefix not protected")
	}
}

func TestBatchAPIAndComments(t *testing.T) {
	h := Handler("/test/", nil)
	parse := httptest.NewRecorder()
	h.ServeHTTP(parse, httptest.NewRequest("POST", "http://localhost/test/api/parse", bytes.NewBufferString(`{"text":"#备注\nA\n/give @a minecraft:diamond 1\n#备注\nB","skip_blank":true,"delay_tenths":3}`)))
	if parse.Code != 200 {
		t.Fatal(parse.Body.String())
	}
	var parsed struct {
		Lines    []circuit.Line `json:"lines"`
		Comments int            `json:"comment_lines"`
	}
	if err := json.Unmarshal(parse.Body.Bytes(), &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.Comments != 2 || len(parsed.Lines) != 3 {
		t.Fatal(parsed)
	}
	gap := 1
	p := project.Project{Request: circuit.Request{Name: "中文剧情", MaxWidth: 8, LaneGap: &gap, Target: "@a", Lines: parsed.Lines}, SplitMode: "count", SplitEvery: 2}
	for _, route := range []string{"preview", "export", "export-part"} {
		body, _ := json.Marshal(struct {
			project.Project
			PartIndex int `json:"part_index"`
		}{p, 1})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("POST", "http://localhost/test/api/"+route, bytes.NewReader(body)))
		if w.Code != 200 {
			t.Fatal(route, w.Body.String())
		}
		if route == "preview" {
			var v project.Preview
			if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil || v.PartIndex != 1 || len(v.Parts) != 2 || len(v.Events) != 1 {
				t.Fatal(v, err)
			}
		}
		if route == "export" {
			if z, err := zip.NewReader(bytes.NewReader(w.Body.Bytes()), int64(w.Body.Len())); err != nil || len(z.File) != 4 {
				t.Fatal(z, err)
			}
		}
		if route == "export-part" {
			_, param, err := mime.ParseMediaType(w.Header().Get("Content-Disposition"))
			if err != nil || param["filename"] != "中文剧情_part_002.schem" {
				t.Fatal(param, err)
			}
		}
	}
	for _, bad := range []string{`{"lines":[],"max_width":8,"target":"@a"}`, `{"unknown":1}`, `{"lines":[{"text":"A"}],"max_width":8,"target":"@a","lane_gap":0}`} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("POST", "http://localhost/test/api/export", bytes.NewBufferString(bad)))
		if w.Code != 400 {
			t.Fatal("invalid request accepted", bad)
		}
	}
}
