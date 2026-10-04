package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"XinText/internal/assets"
	"XinText/internal/runcmd"
)

// ExportService renders markdown to HTML / DOCX / MD. HTML and DOCX shell out
// to a local pandoc binary; TXT export is implemented in the frontend (no
// pandoc needed). PDF export (ExportPDFFromHTML) does not use pandoc:
// the frontend renders the document with Vditor.preview and Microsoft Edge
// headless prints it to PDF, matching the window.print() output exactly.
type ExportService struct {
	config   *ConfigService
	vditorFS fs.FS // 嵌入的 vditor 静态资源子树（CSS、KaTeX 字体），由 main.go 注入
}

// NewExportService creates a new ExportService. The config service is used to
// honour a user-configured pandoc location set in the settings dialog.
// vditorFS 是嵌入的 vditor 静态资源子树（CSS、KaTeX 字体），可为 nil；
// 通过构造函数注入而非导出 setter，避免被 wails 当作前端可调用方法
// （fs.FS 接口无法 JSON 序列化）。
func NewExportService(cfg *ConfigService, vditorFS fs.FS) *ExportService {
	return &ExportService{config: cfg, vditorFS: vditorFS}
}

// PandocInfo reports whether pandoc is available and its version string.
type PandocInfo struct {
	Available bool   `json:"available"`
	Path      string `json:"path"`
	Version   string `json:"version"`
}

// resolvePandoc locates the pandoc executable. Resolution order:
//  1. XinText_PANDOC env var (explicit override)
//  2. the user-configured pandoc location (settings dialog)
//  3. a pandoc binary bundled next to the application executable
//  4. pandoc on PATH
//  5. a Windows development fallback
func (s *ExportService) resolvePandoc() (string, error) {
	if p := os.Getenv("XinText_PANDOC"); p != "" {
		if fileExists(p) {
			return p, nil
		}
	}
	if s.config != nil {
		if cfg, err := s.config.GetConfig(); err == nil && cfg.PandocPath != "" {
			if fileExists(cfg.PandocPath) {
				return cfg.PandocPath, nil
			}
		}
	}
	if p := bundledPandocPath(); p != "" {
		return p, nil
	}
	if p, err := exec.LookPath("pandoc"); err == nil {
		return p, nil
	}
	return "", fmt.Errorf("pandoc not found: set XinText_PANDOC, bundle it next to the app, or add pandoc to PATH")
}

// bundledPandocPath returns the path to a pandoc binary shipped next to the
// running executable ("./pandoc/pandoc[.exe]"), or "" when not bundled.
func bundledPandocPath() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	dir := filepath.Dir(exe)
	name := "pandoc"
	if runtime.GOOS == "windows" {
		name = "pandoc.exe"
	}
	candidates := []string{
		filepath.Join(dir, name),
		filepath.Join(dir, "pandoc", name),
		filepath.Join(dir, "bin", name),
	}
	for _, c := range candidates {
		if fileExists(c) {
			return c
		}
	}
	return ""
}

// CheckPandoc returns the pandoc availability and version string.
func (s *ExportService) CheckPandoc() (PandocInfo, error) {
	info := PandocInfo{}
	path, err := s.resolvePandoc()
	if err != nil {
		return info, nil
	}
	info.Available = true
	info.Path = path
	// 10s 超时兜底；隐藏窗口防止 cmd 黑框闪烁
	out, err := runcmd.Run(nil, 10*time.Second, path, "--version")
	if err == nil {
		firstLine := strings.SplitN(string(out), "\n", 2)[0]
		info.Version = strings.TrimSpace(firstLine)
	}
	return info, nil
}

// CheckPandocPath validates a specific pandoc executable chosen by the user:
// it runs `pandoc --version` on it and reports availability/version.
func (s *ExportService) CheckPandocPath(path string) (PandocInfo, error) {
	info := PandocInfo{}
	if path == "" || !fileExists(path) {
		return info, nil
	}
	out, err := runcmd.Run(nil, 10*time.Second, path, "--version")
	if err != nil {
		return info, nil
	}
	info.Available = true
	info.Path = path
	info.Version = strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	return info, nil
}

// ExportHTML converts markdown to a standalone HTML document written to
// outPath. Images are resolved relative to workDir (usually the document
// directory).
func (s *ExportService) ExportHTML(markdown, workDir, outPath string) error {
	pandocPath, err := s.resolvePandoc()
	if err != nil {
		return err
	}
	cssPath, err := writeCSSHeader()
	if err != nil {
		return fmt.Errorf("failed to create CSS header: %w", err)
	}
	defer os.Remove(cssPath)
	args := htmlArgs(outPath, cssPath)
	return runPandoc(pandocPath, args, workDir, markdown)
}

// ExportDOCX converts markdown to a Word document at outPath. Requires only pandoc.
func (s *ExportService) ExportDOCX(markdown, workDir, outPath string) error {
	pandocPath, err := s.resolvePandoc()
	if err != nil {
		return err
	}
	return runPandoc(pandocPath, docxArgs(outPath), workDir, markdown)
}

// mdImageRefRe matches markdown image syntax ![alt](path).
var mdImageRefRe = regexp.MustCompile(`!\[([^\]]*)\]\(([^)\n]+)\)`)

// ExportMD writes markdown to outPath with every local image embedded as a
// base64 data URL, producing a single self-contained .md file. Remote URLs
// and existing data URLs are kept unchanged; unreadable local files keep
// their original reference (graceful degradation).
func (s *ExportService) ExportMD(markdown, workDir, outPath string) error {
	embedded := embedMarkdownImages(markdown, workDir)
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	if err := os.WriteFile(outPath, []byte(embedded), 0644); err != nil {
		return fmt.Errorf("failed to write markdown: %w", err)
	}
	return nil
}

// embedMarkdownImages replaces local image references in markdown with
// base64 data URLs. Relative references are resolved against workDir.
func embedMarkdownImages(markdown, workDir string) string {
	return mdImageRefRe.ReplaceAllStringFunc(markdown, func(match string) string {
		sub := mdImageRefRe.FindStringSubmatch(match)
		if len(sub) != 3 {
			return match
		}
		alt, ref := sub[1], strings.TrimSpace(sub[2])
		// Keep remote URLs and already-embedded data URLs untouched.
		lower := strings.ToLower(ref)
		if strings.HasPrefix(lower, "data:") ||
			strings.HasPrefix(lower, "http://") ||
			strings.HasPrefix(lower, "https://") {
			return match
		}
		// Drop an optional title part: ![alt](path "title")
		path := strings.TrimSpace(strings.Fields(ref)[0])
		// Strip angle-bracket wrapped paths: ![alt](<C:/my pic.png>)
		path = strings.TrimPrefix(path, "<")
		path = strings.TrimSuffix(path, ">")

		absPath := path
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(workDir, absPath)
		}
		data, err := os.ReadFile(absPath)
		if err != nil {
			// 容错：文件不可读时保留原始引用
			return match
		}
		mime := imageMIMEByPath(absPath)
		return fmt.Sprintf("![%s](data:%s;base64,%s)",
			alt, mime, base64.StdEncoding.EncodeToString(data))
	})
}

// imageMIMEByPath maps an image file extension to its MIME type.
func imageMIMEByPath(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".bmp":
		return "image/bmp"
	default:
		return "application/octet-stream"
	}
}

// ExportFile converts an already-saved markdown file by path.
// format is "html", "docx" or "md" (TXT export is implemented in the frontend
// and never reaches here). The document is passed to pandoc
// as an input file with its own directory as working directory, so large
// documents and their image resources never cross the frontend<->Go bridge as
// strings. PDF is not supported here: it goes through ExportPDFFromHTML
// (Microsoft Edge headless) instead.
func (s *ExportService) ExportFile(mdPath, outPath, format string) error {
	mdPath, err := filepath.Abs(mdPath)
	if err != nil {
		return fmt.Errorf("invalid document path: %w", err)
	}
	if !fileExists(mdPath) {
		return fmt.Errorf("document not found: %s", mdPath)
	}

	// Self-contained markdown with base64 images needs no pandoc.
	if strings.ToLower(format) == "md" {
		data, err := os.ReadFile(mdPath)
		if err != nil {
			return fmt.Errorf("failed to read document: %w", err)
		}
		return s.ExportMD(string(data), filepath.Dir(mdPath), outPath)
	}

	pandocPath, err := s.resolvePandoc()
	if err != nil {
		return err
	}

	var args []string
	cssPath := ""
	switch strings.ToLower(format) {
	case "html":
		cssPath, err = writeCSSHeader()
		if err != nil {
			return fmt.Errorf("failed to create CSS header: %w", err)
		}
		defer os.Remove(cssPath)
		args = htmlArgs(outPath, cssPath)
	case "docx":
		args = docxArgs(outPath)
	case "txt":
		return fmt.Errorf("TXT export is implemented in the frontend, not via pandoc")
	case "pdf":
		return fmt.Errorf("PDF export is handled by ExportPDFFromHTML (Microsoft Edge headless), not pandoc")
	default:
		return fmt.Errorf("unsupported export format: %s", format)
	}

	// Input file last; pandoc reads it from disk. Empty stdin marker.
	args = append(args, mdPath)
	return runPandoc(pandocPath, args, filepath.Dir(mdPath), "")
}

func htmlArgs(outPath, cssHeaderPath string) []string {
	args := []string{
		"-f", "markdown",
		"-t", "html5",
		"-s",
		"--embed-resources",
		"--toc",
		"--toc-depth=6",
		"--metadata", "title=XinText Export",
		"-o", outPath,
	}
	if cssHeaderPath != "" {
		args = append(args, "-H", cssHeaderPath)
	}
	return args
}

// modernHTMLCSS is injected into exported HTML documents for a clean,
// modern reading experience. Matches the editor's light theme.
// Includes a right-side TOC sidebar with scroll-spy, similar to read mode.
const modernHTMLCSS = `<style>
:root {
  --text: #303133;
  --muted: #909399;
  --border: #e4e7ed;
  --bg: #ffffff;
  --code-bg: #f5f7fa;
  --link: #409eff;
  --heading: #303133;
  --toc-w: 240px;
}
* { box-sizing: border-box; }
body {
  margin: 0 !important;
  padding: 0 !important;
  max-width: none !important;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC",
    "Microsoft YaHei", "Helvetica Neue", Arial, sans-serif;
  font-size: 16px;
  line-height: 1.8;
  color: var(--text);
  background: var(--bg);
}
/* Hide pandoc's default title block (replaced by .brand-bar) */
#title-block-header { display: none !important; }
/* Main content: single container, shifted right for left-side TOC */
.body-content {
  max-width: 760px;
  margin: 0 auto 0 var(--toc-w);
  padding: 48px 32px 80px;
  min-height: calc(100vh - 48px);
}
h1, h2, h3, h4, h5, h6 {
  color: var(--heading);
  font-weight: 700;
  line-height: 1.4;
  margin-top: 1.8em;
  margin-bottom: 0.6em;
  scroll-margin-top: 20px;
}
h1 { font-size: 2em; border-bottom: 2px solid var(--border); padding-bottom: 0.3em; margin-top: 0; }
h2 { font-size: 1.5em; border-bottom: 1px solid var(--border); padding-bottom: 0.2em; }
h3 { font-size: 1.25em; }
h4 { font-size: 1.05em; }
p { margin: 0.8em 0; }
a { color: var(--link); text-decoration: none; transition: color 0.15s; }
a:hover { text-decoration: underline; }
blockquote {
  margin: 1em 0;
  padding: 8px 16px;
  border-left: 4px solid var(--link);
  background: var(--code-bg);
  border-radius: 0 4px 4px 0;
  color: var(--muted);
}
blockquote p { margin: 0.4em 0; }
code {
  font-family: Consolas, "SFMono-Regular", Menlo, Monaco, "Courier New", monospace;
  font-size: 0.88em;
  background: var(--code-bg);
  padding: 2px 6px;
  border-radius: 4px;
  color: #c7254e;
}
pre {
  background: var(--code-bg);
  border-radius: 8px;
  padding: 16px 20px;
  overflow-x: auto;
  line-height: 1.6;
}
pre code {
  background: none;
  padding: 0;
  border-radius: 0;
  color: var(--text);
  font-size: 0.85em;
}
table {
  border-collapse: collapse;
  width: 100%;
  margin: 1em 0;
  font-size: 0.92em;
}
th, td {
  border: 1px solid var(--border);
  padding: 8px 12px;
}
th { background: var(--code-bg); font-weight: 600; }
tr:nth-child(even) td { background: rgba(0,0,0,0.02); }
img { max-width: 100%; border-radius: 8px; }
hr { border: none; border-top: 1px solid var(--border); margin: 2em 0; }
ul, ol { padding-left: 1.8em; }
li { margin: 0.3em 0; }

/* ---- Left-side TOC sidebar (pandoc generates <nav id="TOC">) ---- */
#TOC {
  position: fixed;
  top: 0;
  left: 0;
  width: var(--toc-w);
  height: 100vh;
  overflow-y: auto;
  padding: 48px 16px 48px 20px;
  border-right: 1px solid var(--border);
  background: var(--bg);
  font-size: 13px;
  transition: transform 0.25s ease;
  z-index: 10;
  display: flex;
  flex-direction: column;
}
/* Search filter input at top of TOC */
.toc-search {
  position: relative;
  margin-bottom: 8px;
  flex-shrink: 0;
}
.toc-search input {
  width: 100%;
  padding: 6px 8px 6px 28px;
  border: 1px solid var(--border);
  border-radius: 6px;
  font-size: 13px;
  outline: none;
  background: var(--code-bg);
  color: var(--text);
  transition: border-color 0.15s;
}
.toc-search input:focus {
  border-color: var(--link);
}
.toc-search input::placeholder { color: var(--muted); }
.toc-search .toc-search-icon {
  position: absolute;
  left: 8px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--muted);
  pointer-events: none;
}
/* TOC list scrolls, search stays fixed */
#TOC > ul {
  list-style: none;
  padding: 0;
  margin: 0;
  flex: 1;
  overflow-y: auto;
}
#TOC ul ul { list-style: none; padding-left: 14px; margin: 0; }
#TOC li { margin: 0; }
#TOC a {
  display: block;
  padding: 4px 8px;
  color: var(--muted);
  border-radius: 4px;
  border-left: 2px solid transparent;
  transition: color 0.15s, background 0.15s, border-color 0.15s;
  line-height: 1.5;
  word-break: break-word;
}
#TOC a:hover {
  color: var(--text);
  background: var(--code-bg);
}
#TOC a.toc-active {
  color: var(--link);
  font-weight: 600;
  border-left-color: var(--link);
  background: rgba(64, 158, 255, 0.06);
}
#TOC ul ul a { font-size: 0.92em; }

/* TOC collapsed state */
body.toc-hidden #TOC { transform: translateX(-100%); }
body.toc-hidden .body-content {
  margin-left: auto;
}

/* Top brand bar: logo+title on left, toggle button on right */
.brand-bar {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 12px 0 16px;
  background: var(--bg);
  border-bottom: 1px solid var(--border);
  z-index: 15;
  font-size: 15px;
  font-weight: 700;
  color: var(--heading);
}
.brand-bar .brand-left {
  display: flex;
  align-items: center;
  gap: 8px;
}
.brand-bar .brand-link {
  display: flex;
  align-items: center;
  gap: 8px;
  text-decoration: none;
  color: inherit;
  border-radius: 4px;
  padding: 2px 4px;
  margin: -2px -4px;
  transition: color 0.15s, background 0.15s;
}
.brand-bar .brand-link:hover {
  color: var(--link);
  background: var(--hover, rgba(0, 0, 0, 0.04));
}
.brand-bar .brand-logo {
  width: 20px;
  height: 20px;
  flex-shrink: 0;
}

/* Toggle button in brand bar right side */
.toc-toggle {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--bg);
  cursor: pointer;
  font-size: 13px;
  color: var(--muted);
  transition: all 0.15s;
  white-space: nowrap;
}
.toc-toggle:hover {
  color: var(--link);
  border-color: var(--link);
}

/* pandoc wraps content in <body>, move the TOC outside the main flow */
#TOC > ul > li > a { font-weight: 500; }
/* Push content below the fixed brand bar */
.body-content { padding-top: 64px; }

/* On narrow screens, hide the TOC sidebar and make content full width */
@media (max-width: 1100px) {
  :root { --toc-w: 0px; }
  #TOC { display: none; }
  .body-content {
    margin: 0 auto;
    max-width: 760px;
  }
}
</style>
<script>
document.addEventListener('DOMContentLoaded', function() {
  var toc = document.getElementById('TOC');

  // ---- Wrap all content (everything except #TOC) in a single .body-content div ----
  var bodyChildren = Array.prototype.slice.call(document.body.children);
  var contentNodes = bodyChildren.filter(function(n) {
    return n !== toc && n.nodeType === 1;
  });
  var contentDiv = document.createElement('div');
  contentDiv.className = 'body-content';
  document.body.insertBefore(contentDiv, toc ? toc.nextSibling : document.body.firstChild);
  contentNodes.forEach(function(n) { contentDiv.appendChild(n); });

  // ---- Brand bar: logo+title on left, toggle button on right ----
  var brand = document.createElement('div');
  brand.className = 'brand-bar';

  var brandLeft = document.createElement('div');
  brandLeft.className = 'brand-left';
  brandLeft.innerHTML = '<a class="brand-link" href="https://gitcode.com/qq_23994787/XinText/releases" target="_blank" rel="noopener noreferrer" title="前往 XinText 发布页"><img class="brand-logo" src="data:image/png;base64,{{LOGO_BASE64}}" alt="XinText" /><span>XinText</span></a>';
  brand.appendChild(brandLeft);

  var btn = document.createElement('button');
  btn.className = 'toc-toggle';
  btn.title = '目录';
  btn.innerHTML = '<svg viewBox="0 0 16 16" width="14" height="14"><path d="M2 3.5h1.5M6 3.5h8M2 8h1.5M6 8h8M2 12.5h1.5M6 12.5h8" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" fill="none"/></svg><span class="toc-toggle-text">折叠目录</span>';
  btn.addEventListener('click', function() {
    var hidden = document.body.classList.toggle('toc-hidden');
    btn.querySelector('.toc-toggle-text').textContent = hidden ? '展开目录' : '折叠目录';
  });
  brand.appendChild(btn);

  document.body.insertBefore(brand, document.body.firstChild);

  if (!toc) return;

  // ---- Search filter input at top of TOC ----
  var searchBox = document.createElement('div');
  searchBox.className = 'toc-search';
  searchBox.innerHTML = '<span class="toc-search-icon"><svg viewBox="0 0 16 16" width="14" height="14"><path d="M7 2a5 5 0 1 0 0 10A5 5 0 0 0 7 2zm0 1.5a3.5 3.5 0 1 1 0 7 3.5 3.5 0 0 1 0-7zM11 11l3 3" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" fill="none"/></svg></span><input type="text" placeholder="搜索目录..." />';
  var searchInput = searchBox.querySelector('input');
  var tocList = toc.querySelector('ul');
  toc.insertBefore(searchBox, tocList);

  // Collect all TOC links for filtering
  var allTocLinks = Array.prototype.slice.call(toc.querySelectorAll('a[href^="#"]'));
  searchInput.addEventListener('input', function() {
    var q = searchInput.value.trim().toLowerCase();
    allTocLinks.forEach(function(a) {
      var text = a.textContent.toLowerCase();
      var match = !q || text.indexOf(q) >= 0;
      var li = a.closest('li');
      if (li) li.style.display = match ? '' : 'none';
    });
    // Show parent <li> of matching children
    if (q) {
      allTocLinks.forEach(function(a) {
        var text = a.textContent.toLowerCase();
        if (text.indexOf(q) >= 0) {
          var li = a.closest('li');
          var p = li ? li.parentElement.closest('li') : null;
          while (p) {
            p.style.display = '';
            p = p.parentElement.closest('li');
          }
        }
      });
    }
  });

  // Build heading list from TOC anchors
  var tocLinks = Array.prototype.slice.call(toc.querySelectorAll('a[href^="#"]'));
  var headings = tocLinks.map(function(a) {
    var id = a.getAttribute('href').slice(1);
    return { id: id, el: document.getElementById(id), link: a };
  }).filter(function(h) { return h.el; });

  // Scroll-spy: highlight the TOC entry for the heading currently in view
  var activeEl = null;
  function updateActive() {
    var current = null;
    for (var i = 0; i < headings.length; i++) {
      var rect = headings[i].el.getBoundingClientRect();
      if (rect.top <= 100) current = headings[i];
      else break;
    }
    if (!current && headings.length) current = headings[0];
    if (current && activeEl !== current.link) {
      if (activeEl) activeEl.classList.remove('toc-active');
      current.link.classList.add('toc-active');
      activeEl = current.link;
      // Scroll TOC to keep active item visible
      var tocRect = toc.getBoundingClientRect();
      var linkRect = current.link.getBoundingClientRect();
      if (linkRect.top < tocRect.top + 40 || linkRect.bottom > tocRect.bottom - 40) {
        current.link.scrollIntoView({ block: 'center', behavior: 'smooth' });
      }
    }
  }

  // Throttled scroll listener
  var ticking = false;
  function onScroll() {
    if (!ticking) {
      requestAnimationFrame(function() { updateActive(); ticking = false; });
      ticking = true;
    }
  }
  window.addEventListener('scroll', onScroll, { passive: true });
  window.addEventListener('resize', onScroll, { passive: true });
  updateActive();

  // Smooth-scroll to heading on TOC link click
  tocLinks.forEach(function(a) {
    a.addEventListener('click', function(e) {
      e.preventDefault();
      var id = a.getAttribute('href').slice(1);
      var el = document.getElementById(id);
      if (el) {
        var top = el.getBoundingClientRect().top + (window.scrollY || document.documentElement.scrollTop) - 12;
        window.scrollTo({ top: top, behavior: 'smooth' });
      }
    });
  });
});
</script>
`

// writeCSSHeader writes the modern CSS to a temp file wrapped in <style> tags
// and returns its path. The logo PNG is embedded as base64. Caller must remove
// the file when done.
func writeCSSHeader() (string, error) {
	css := modernHTMLCSS

	// Logo is embedded at build time via //go:embed (internal/assets/logo.png)
	// so it ships inside the binary and doesn't depend on the runtime working
	// directory. Previously using filepath.Join("internal","assets","logo.png")
	// failed after packaging because the CWD wasn't the project root, producing
	// an empty `data:image/png;base64,` src.
	logoB64 := base64.StdEncoding.EncodeToString(assets.LogoPNG)
	css = strings.ReplaceAll(css, "{{LOGO_BASE64}}", logoB64)

	f, err := os.CreateTemp("", "XinText-css-*.html")
	if err != nil {
		return "", err
	}
	if _, err := f.WriteString(css); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

func docxArgs(outPath string) []string {
	return []string{
		"-f", "markdown",
		"-t", "docx",
		"-o", outPath,
	}
}

// runPandoc runs pandoc. When markdown is non-empty it is fed on stdin;
// an empty string means the input file is already present as the last
// positional argument (path-based export), so nothing is written to stdin.
func runPandoc(bin string, args []string, workDir, markdown string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// 2 分钟超时兜底；隐藏窗口防止导出时 cmd 黑框闪烁
	cmd := runcmd.Command(ctx, bin, args...)
	if workDir != "" {
		cmd.Dir = workDir
	}
	if markdown != "" {
		cmd.Stdin = strings.NewReader(markdown)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("pandoc failed: %s", msg)
	}
	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// printPDFCSS 注入到导出 HTML 的 <head>，与前端 usePrint.ts 的 PRINT_CSS 保持一致
// （仅打印相关样式，不含 .XinText-print-root 容器规则）。@page 保留默认 14mm/16mm
// 页边距，配合 msedge --no-pdf-header-footer 隐藏默认页眉页脚（时间、URL、标题），
// 仅保留正文，与前端打印路径效果一致。
const printPDFCSS = `
body { margin: 0; padding: 14mm 16mm; background: #fff; color: #303133; }
.vditor-reset { width: 100%; }
h1, h2, h3, h4, h5, h6 { page-break-after: avoid; break-after: avoid; }
pre, table, img, blockquote { page-break-inside: avoid; break-inside: avoid; }
pre { white-space: pre-wrap; word-wrap: break-word; }
img { max-width: 100%; }
table { width: 100%; border-collapse: collapse; }
th, td { border: 1px solid #999; padding: 4px 8px; }
a { color: #000; text-decoration: underline; }
@page { margin: 14mm 16mm; }
`

// chromiumBrowserPath 探测系统已安装的 Chromium 系浏览器（Edge / Chrome /
// Chromium）可执行文件路径。PDF 导出依赖 Chromium headless 的 --print-to-pdf，
// 三者渲染行为一致，可互换。Windows 优先按安装目录找 Edge/Chrome，macOS 按
// /Applications 标准 .app 路径查找，最后统一回退 PATH 搜索。返回 "" 表示未找到。
func chromiumBrowserPath() string {
	var candidates []string
	switch runtime.GOOS {
	case "windows":
		candidates = []string{
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "Microsoft", "Edge", "Application", "msedge.exe"),
			filepath.Join(os.Getenv("ProgramFiles"), "Microsoft", "Edge", "Application", "msedge.exe"),
			filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "Edge", "Application", "msedge.exe"),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("ProgramFiles"), "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("LOCALAPPDATA"), "Google", "Chrome", "Application", "chrome.exe"),
		}
	case "darwin":
		candidates = []string{
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		}
	}
	for _, c := range candidates {
		if c != "" && fileExists(c) {
			return c
		}
	}
	for _, name := range []string{
		"microsoft-edge", "microsoft-edge-stable", "msedge",
		"google-chrome", "google-chrome-stable", "chrome",
		"chromium", "chromium-browser",
	} {
		if p, err := exec.LookPath(name); err == nil && p != "" {
			return p
		}
	}
	return ""
}

// extractFS 把 fs.FS 全量解压到 dest 目录，保留相对路径结构。
// 用于把嵌入的 vditor 静态资源（CSS、KaTeX 字体）写到临时目录供 msedge headless 通过 file:// 加载。
func extractFS(fsys fs.FS, dest string) error {
	return fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}

// ExportPDFFromHTML 用 Chromium 系浏览器（Edge / Chrome / Chromium）headless
// 把已渲染好的 HTML 渲染为 PDF。与前端 window.print() 路径共享同一份 HTML 渲染
// 产物（Vditor.preview 输出 + resolveImagesIn 把图片转 data URL），因此公式
// （KaTeX）、代码高亮、表格样式与打印效果完全一致。
//
// 步骤：
//  1. 创建临时目录，把嵌入的 vditor 静态资源（CSS、KaTeX 字体）解压到 tmpdir/vditor/
//  2. 写 tmpdir/print.html：<link href="vditor/dist/index.css"> + 打印 CSS + 前端传入的 body
//  3. browser --headless=new --no-pdf-header-footer --print-to-pdf=<outPath> file:///tmpdir/print.html
//
// Chromium 渲染引擎与 WebView2 同源，公式（KaTeX）/代码高亮/表格完美渲染。
func (s *ExportService) ExportPDFFromHTML(htmlBody, outPath string) error {
	browser := chromiumBrowserPath()
	if browser == "" {
		return fmt.Errorf("no Chromium-based browser found; install Microsoft Edge, Google Chrome or Chromium to export PDF")
	}
	tmpDir, err := os.MkdirTemp("", "XinText-pdf-html-")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// 解压 vditor 静态资源（CSS、KaTeX 字体）到临时目录，HTML 通过相对路径引用
	if s.vditorFS != nil {
		if err := extractFS(s.vditorFS, filepath.Join(tmpDir, "vditor")); err != nil {
			return fmt.Errorf("extract vditor assets: %w", err)
		}
	}

	// 组装自包含 HTML：link 引用 vditor/dist/index.css，body 是前端 Vditor.preview 渲染产物
	fullHTML := `<!doctype html><html><head><meta charset="utf-8">` +
		`<link rel="stylesheet" href="vditor/dist/index.css">` +
		`<style>` + printPDFCSS + `</style>` +
		`</head><body class="vditor-reset">` + htmlBody + `</body></html>`
	htmlPath := filepath.Join(tmpDir, "print.html")
	if err := os.WriteFile(htmlPath, []byte(fullHTML), 0644); err != nil {
		return fmt.Errorf("write html: %w", err)
	}

	// file:// URL（Windows 路径反斜杠转正斜杠，盘符前补三斜杠）
	fileURL := "file:///" + strings.ReplaceAll(htmlPath, "\\", "/")
	cmd := runcmd.Command(context.Background(), browser,
		"--headless=new",
		"--disable-gpu",
		"--no-pdf-header-footer",
		"--print-to-pdf="+outPath,
		"--virtual-time-budget=10000",
		fileURL,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("chromium print to pdf failed: %w; stderr: %s", err, stderr.String())
	}
	if !fileExists(outPath) {
		return fmt.Errorf("chromium did not produce PDF; stderr: %s", stderr.String())
	}
	return nil
}
