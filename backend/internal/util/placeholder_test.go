package util

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlaceholderPDF(t *testing.T) {
	pdf := PlaceholderPDF([]string{"LexCase Preset Document (placeholder)", "Document ID: 1", "Type: complaint"})
	s := string(pdf)
	if !strings.HasPrefix(s, "%PDF-1.4") {
		t.Error("placeholder pdf should start with %PDF-1.4 header")
	}
	if !strings.HasSuffix(s, "%%EOF\n") {
		t.Error("placeholder pdf should end with EOF marker")
	}
	if !strings.Contains(s, "xref") || !strings.Contains(s, "trailer") || !strings.Contains(s, "startxref") {
		t.Error("placeholder pdf should contain xref table and trailer")
	}
	if !strings.Contains(s, "(Document ID: 1) Tj") {
		t.Error("placeholder pdf should embed text lines")
	}
	// 相同输入必须生成相同字节（重复启动不产生不同文件）。
	if string(PlaceholderPDF([]string{"a", "b"})) != string(PlaceholderPDF([]string{"a", "b"})) {
		t.Error("placeholder pdf should be deterministic")
	}
}

func TestPDFEscape(t *testing.T) {
	if got := pdfEscape(`a(b)c\d`); got != `a\(b\)c\\d` {
		t.Errorf("pdfEscape = %q", got)
	}
	// 非 ASCII（如中文标题）以 ? 代替，不产生非法字节。
	if got := pdfEscape("民事起诉状"); got != "?????" {
		t.Errorf("pdfEscape non-ascii = %q, want ?????", got)
	}
}

func TestEnsureFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cases", "doc1.pdf")

	created, err := EnsureFile(path, []byte("v1"))
	if err != nil || !created {
		t.Fatalf("first EnsureFile created=%v err=%v, want created=true", created, err)
	}
	raw, _ := os.ReadFile(path)
	if string(raw) != "v1" {
		t.Fatalf("file content = %q, want v1", raw)
	}

	// 重复执行：不重复生成、不覆盖已有内容。
	if err := os.WriteFile(path, []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	created, err = EnsureFile(path, []byte("v2"))
	if err != nil || created {
		t.Fatalf("second EnsureFile created=%v err=%v, want created=false", created, err)
	}
	raw, _ = os.ReadFile(path)
	if string(raw) != "changed" {
		t.Fatalf("EnsureFile should not overwrite existing file, got %q", raw)
	}
}
