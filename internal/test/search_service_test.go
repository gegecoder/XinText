package test

import (
	"os"
	"path/filepath"
	"testing"

	"XinText/internal/service"
)

// writeSearchFixture 在临时目录写入一个 md 文件并返回目录路径。
func writeSearchFixture(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return dir
}

// TestSearchChineseReturnsAllMatches 中文关键词在一行内多次出现时，
// 所有匹配位置都必须返回，且 rune 坐标与片段字符严格对齐。
func TestSearchChineseReturnsAllMatches(t *testing.T) {
	dir := writeSearchFixture(t, "a.md", "编辑器测试：编辑器编辑器连续出现，Markdown编辑器结尾\n")

	hits, err := service.NewSearchService().SearchInDirectory(dir, "编辑器", "md")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("want 1 hit, got %d", len(hits))
	}
	runes := []rune(hits[0].Snippet)
	want := 4 // “编辑器”共出现 4 次
	if got := len(hits[0].Matches); got != want {
		t.Fatalf("want %d match ranges, got %d: %+v", want, got, hits[0].Matches)
	}
	for _, m := range hits[0].Matches {
		if m.End > len(runes) || m.Start < 0 || m.End-m.Start != 3 {
			t.Fatalf("invalid range %+v (snippet rune len=%d)", m, len(runes))
		}
		if string(runes[m.Start:m.End]) != "编辑器" {
			t.Fatalf("range %+v points to %q, want 编辑器", m, string(runes[m.Start:m.End]))
		}
	}
}

// TestSearchAndSemantics 多关键词 AND：缺任一词不命中；都存在时返回
// 各自全部位置（重叠区间已合并）。
func TestSearchAndSemantics(t *testing.T) {
	dir := writeSearchFixture(t, "b.md",
		"Go 语言 Go, Go\n缺少另一个关键词的行\nGo 和 Markdown 与 Go\n")

	hits, err := service.NewSearchService().SearchInDirectory(dir, "go markdown", "all")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("want 1 hit (AND semantics), got %d: %+v", len(hits), hits)
	}
	if hits[0].LineNumber != 3 {
		t.Fatalf("want line 3, got %d", hits[0].LineNumber)
	}
	runes := []rune(hits[0].Snippet)
	var hitMarkdown bool
	for _, m := range hits[0].Matches {
		switch string(runes[m.Start:m.End]) {
		case "Go", "Markdown":
			if s := string(runes[m.Start:m.End]); s == "Markdown" {
				hitMarkdown = true
			}
		default:
			t.Fatalf("unexpected matched text %q at %+v", string(runes[m.Start:m.End]), m)
		}
	}
	if !hitMarkdown {
		t.Fatalf("Markdown range missing in %+v", hits[0].Matches)
	}
}

// TestSearchSnippetOffsetsWithEllipsis 长行截断后，片段带前导省略号，
// 返回的坐标必须相对片段（含省略号占位）仍然精确。
func TestSearchSnippetOffsetsWithEllipsis(t *testing.T) {
	// 构造超过 80 rune 的长行，关键词出现在靠后位置
	line := ""
	for i := 0; i < 60; i++ {
		line += "很"
	}
	line += "关键词"
	for i := 0; i < 40; i++ {
		line += "长"
	}
	dir := writeSearchFixture(t, "long.md", line+"\n")

	hits, err := service.NewSearchService().SearchInDirectory(dir, "关键词", "md")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) != 1 || len(hits[0].Matches) != 1 {
		t.Fatalf("want 1 hit/1 match, got %+v", hits)
	}
	runes := []rune(hits[0].Snippet)
	m := hits[0].Matches[0]
	if string(runes[m.Start:m.End]) != "关键词" {
		t.Fatalf("offset misaligned after ellipsis: %q (range %+v)", string(runes[m.Start:m.End]), m)
	}
}
