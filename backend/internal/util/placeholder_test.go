package util

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
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

func TestEnsureFileConcurrent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cases", "doc.pdf")
	content := []byte("deterministic-content")

	// 多实例并发执行：只能得到一个完整文件，且恰好一个写入者生效。
	const n = 16
	var wg sync.WaitGroup
	created := make([]bool, n)
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			created[i], errs[i] = EnsureFile(path, content)
		}(i)
	}
	wg.Wait()
	createdCount := 0
	for i := 0; i < n; i++ {
		if errs[i] != nil {
			t.Fatalf("concurrent EnsureFile[%d] error: %v", i, errs[i])
		}
		if created[i] {
			createdCount++
		}
	}
	if createdCount != 1 {
		t.Errorf("created count = %d, want exactly 1", createdCount)
	}
	raw, err := os.ReadFile(path)
	if err != nil || string(raw) != string(content) {
		t.Fatalf("final content = %q, err=%v, want %q", raw, err, content)
	}

	// 内容不同的并发写入：最终为其中一份完整内容，且已有内容不被改写。
	path2 := filepath.Join(dir, "cases", "doc2.pdf")
	contents := [][]byte{[]byte("content-A"), []byte("content-B"), []byte("content-C")}
	for i := 0; i < len(contents); i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _ = EnsureFile(path2, contents[i])
		}(i)
	}
	wg.Wait()
	final, err := os.ReadFile(path2)
	if err != nil {
		t.Fatal(err)
	}
	valid := false
	for _, c := range contents {
		if string(final) == string(c) {
			valid = true
		}
	}
	if !valid {
		t.Errorf("final content %q is not one of the complete candidates", final)
	}
	if c, _ := EnsureFile(path2, []byte("rewrite")); c {
		t.Error("EnsureFile should not rewrite existing file")
	}
	final2, _ := os.ReadFile(path2)
	if string(final2) != string(final) {
		t.Error("existing file content was rewritten")
	}

	// 临时文件不残留。
	matches, _ := filepath.Glob(filepath.Join(dir, "cases", ".ensure-*"))
	if len(matches) != 0 {
		t.Errorf("temp files leaked: %v", matches)
	}
}
