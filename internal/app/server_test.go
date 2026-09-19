package app

import (
	"bytes"
	"dialogueforge/internal/circuit"
	"encoding/json"
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
