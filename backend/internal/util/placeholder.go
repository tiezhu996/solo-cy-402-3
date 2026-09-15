package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EnsureFile 文件不存在时写入给定内容，已存在则跳过（幂等）。
// 返回是否真正写入了文件；重复调用不产生重复文件，也不覆盖已有内容。
func EnsureFile(path string, content []byte) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, fmt.Errorf("stat file: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, fmt.Errorf("create file dir: %w", err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return false, fmt.Errorf("write file: %w", err)
	}
	return true, nil
}

// PlaceholderPDF 生成最小单页 PDF（仅 ASCII 文本），用于补齐预置文档的缺失文件。
// 内容确定性：相同输入永远生成相同字节，重复启动不会产生不同文件。
func PlaceholderPDF(lines []string) []byte {
	var content strings.Builder
	content.WriteString("BT /F1 14 Tf 72 760 Td 22 TL ")
	for i, ln := range lines {
		if i > 0 {
			content.WriteString("T* ")
		}
		content.WriteString("(" + pdfEscape(ln) + ") Tj ")
	}
	content.WriteString("ET")
	stream := content.String()

	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream),
	}
	var b strings.Builder
	b.WriteString("%PDF-1.4\n")
	offsets := make([]int, 0, len(objects))
	for i, obj := range objects {
		offsets = append(offsets, b.Len())
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xrefStart := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n", len(objects)+1)
	b.WriteString("0000000000 65535 f \n")
	for _, off := range offsets {
		fmt.Fprintf(&b, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xrefStart)
	return []byte(b.String())
}

// pdfEscape 转义 PDF 文本对象；Helvetica 不支持 CJK，非 ASCII 字符以 '?' 代替。
func pdfEscape(s string) string {
	var r strings.Builder
	for _, ch := range s {
		switch {
		case ch == '(' || ch == ')' || ch == '\\':
			r.WriteByte('\\')
			r.WriteByte(byte(ch))
		case ch >= 32 && ch <= 126:
			r.WriteByte(byte(ch))
		default:
			r.WriteByte('?')
		}
	}
	return r.String()
}
