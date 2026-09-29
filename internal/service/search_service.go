package service

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

// TextMatch 描述一条命中中单个关键词的位置区间（rune 下标，左闭右开），
// 坐标相对于 SearchHit.Snippet。
type TextMatch struct {
	Start int `json:"start"` // 起始 rune 下标（含）
	End   int `json:"end"`   // 结束 rune 下标（不含）
}

// SearchHit 表示一条全文检索命中（文件中的一行）。
type SearchHit struct {
	Path       string      `json:"path"`       // 文件绝对路径
	Name       string      `json:"name"`       // 文件名
	LineNumber int         `json:"lineNumber"` // 行号（从 1 开始）
	Snippet    string      `json:"snippet"`    // 命中行去空白后的片段
	Matches    []TextMatch `json:"matches"`    // 片段内全部命中位置（多关键词、多次出现）
}

// SearchService 在已打开目录内对文档做关键字全文检索。
type SearchService struct{}

func NewSearchService() *SearchService {
	return &SearchService{}
}

// 单次检索最多返回的命中条数，防止超大目录拖垮界面
const maxSearchHits = 300

// SearchInDirectory 递归检索 dir 下的文档，返回命中行。
// keyword 支持空格分隔的多个关键词，采用 AND 语义（一行包含全部关键词
// 才算命中，大小写不敏感）。extFilter: "md"（.md/.markdown）、"txt"
// （.txt）、其他值（含 "all"）表示同时检索 md 与 txt。
func (s *SearchService) SearchInDirectory(dir, keyword, extFilter string) ([]SearchHit, error) {
	parts := strings.Fields(keyword)
	if dir == "" || len(parts) == 0 {
		return []SearchHit{}, nil
	}
	kwLowers := make([]string, 0, len(parts))
	for _, p := range parts {
		kwLowers = append(kwLowers, strings.ToLower(p))
	}

	allow := func(name string) bool {
		ext := strings.ToLower(filepath.Ext(name))
		switch extFilter {
		case "md":
			return ext == ".md" || ext == ".markdown"
		case "txt":
			return ext == ".txt"
		default: // all
			return ext == ".md" || ext == ".markdown" || ext == ".txt"
		}
	}

	hits := make([]SearchHit, 0, 64)
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || len(hits) >= maxSearchHits {
			return nil
		}
		name := d.Name()
		if strings.HasPrefix(name, ".") || !allow(name) {
			return nil
		}
		fileHits, herr := scanFile(path, name, kwLowers)
		if herr == nil {
			hits = append(hits, fileHits...)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(hits) > maxSearchHits {
		hits = hits[:maxSearchHits]
	}
	return hits, nil
}

// byteRange 是整行文本上的字节区间（左闭右开）。
type byteRange struct {
	start int
	end   int
}

// scanFile 按行扫描单个文件，命中行需包含全部关键词（AND）。
// 每个关键词在一行内的全部出现位置都会被收集并返回，不再限制单文件条数
// （总体上限由 maxSearchHits 控制），确保文本中所有匹配位置都能呈现。
func scanFile(path, name string, kwLowers []string) ([]SearchHit, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	hits := make([]SearchHit, 0, 4)
	scanner := bufio.NewScanner(f)
	// 单行最大 1MB，避免超长行导致扫描失败
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		trimmed := strings.TrimSpace(scanner.Text())
		if trimmed == "" {
			continue
		}
		// 在整行上定位全部关键词的全部出现位置（基于字节下标，UTF-8 安全：
		// 中文等多字节字符的续字节不会与 ASCII 关键词发生误匹配）
		ranges, ok := matchRanges(trimmed, kwLowers)
		if !ok {
			continue
		}
		snippet, winStart, winEnd, prefixRunes := buildSnippet(trimmed, ranges)
		matches := rangesToWindow(ranges, trimmed, winStart, winEnd, prefixRunes, utf8.RuneCountInString(snippet))
		hits = append(hits, SearchHit{
			Path:       path,
			Name:       name,
			LineNumber: lineNo,
			Snippet:    snippet,
			Matches:    matches,
		})
	}
	return hits, scanner.Err()
}

// matchRanges 返回一行中所有关键词的全部命中字节区间（已按位置排序并合并
// 重叠区间）；任一关键词一次都未出现时 ok 为 false（AND 语义）。
func matchRanges(line string, kwLowers []string) (ranges []byteRange, ok bool) {
	lower := strings.ToLower(line)
	ranges = make([]byteRange, 0, 4)
	for _, kw := range kwLowers {
		from := 0
		found := false
		for from <= len(lower) {
			idx := strings.Index(lower[from:], kw)
			if idx < 0 {
				break
			}
			start := from + idx
			ranges = append(ranges, byteRange{start: start, end: start + len(kw)})
			// 前进一个字节即可，允许重叠匹配（如 "aa" 匹配 "aaa" 两次）
			from = start + 1
			found = true
		}
		if !found {
			return nil, false
		}
	}
	sort.Slice(ranges, func(i, j int) bool { return ranges[i].start < ranges[j].start })
	// 仅合并真正重叠的区间（多关键词互为子串时）；首尾相接不算重叠，
	// 例如“编辑器编辑器”应保留为两个独立命中位置
	merged := ranges[:0]
	for _, r := range ranges {
		if n := len(merged); n > 0 && r.start < merged[n-1].end {
			if r.end > merged[n-1].end {
				merged[n-1].end = r.end
			}
			continue
		}
		merged = append(merged, r)
	}
	return merged, true
}

// rangesToWindow 把整行字节区间换算成片段窗口内的 rune 区间，丢弃完全落在
// 窗口外的命中，并裁剪跨界命中。prefixRunes 为片段开头省略号占用的 rune 数。
func rangesToWindow(ranges []byteRange, line string, winStart, winEnd, prefixRunes, snippetLen int) []TextMatch {
	matches := make([]TextMatch, 0, len(ranges))
	for _, r := range ranges {
		rs := utf8.RuneCountInString(line[:r.start])
		re := utf8.RuneCountInString(line[:r.end])
		if re <= winStart || rs >= winEnd {
			continue // 完全在片段窗口之外
		}
		start := rs - winStart + prefixRunes
		end := re - winStart + prefixRunes
		if start < 0 {
			start = 0
		}
		if end > snippetLen {
			end = snippetLen
		}
		if end > start {
			matches = append(matches, TextMatch{Start: start, End: end})
		}
	}
	return matches
}

// buildSnippet 在片段过长时以最早的命中位置为基准截断并加省略号（rune 安全，
// 不能按字节切——中文一个字 3 字节，切在中间会乱码）。
// 返回片段文本以及它在原行 rune 序列上的窗口 [winStart, winEnd)，还有开头
// 省略号占用的 rune 数（用于命中坐标换算）。
func buildSnippet(line string, ranges []byteRange) (snippet string, winStart, winEnd, prefixRunes int) {
	const maxLen = 80
	const ctxBefore = 12
	runes := []rune(line)
	total := len(runes)
	if total <= maxLen {
		return line, 0, total, 0
	}
	firstRune := utf8.RuneCountInString(line[:ranges[0].start])
	start := firstRune - ctxBefore
	if start < 0 {
		start = 0
	}
	end := start + maxLen
	if end > total {
		end = total
		start = end - maxLen
		if start < 0 {
			start = 0
		}
	}
	snippet = string(runes[start:end])
	if start > 0 {
		snippet = "…" + snippet
		prefixRunes = 1
	}
	if end < total {
		snippet += "…"
	}
	return snippet, start, end, prefixRunes
}
