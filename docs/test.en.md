# Markdown Full-Feature Test Document

> This document tests how the XinText editor renders various Markdown syntaxes, covering headings, paragraphs, emphasis, lists, blockquotes, code, links, images, tables, task lists, footnotes, math formulas, emoji, inline HTML, and more. Mixed-language test: Mixed English and 中文 Content, numbers 1234567890.

## 1. Headings

# Heading Level 1 H1

## Heading Level 2 H2

### Heading Level 3 H3

#### Heading Level 4 H4

##### Heading Level 5 H5

###### Heading Level 6 H6

## 2. Paragraphs and Line Breaks

This is an ordinary paragraph. It contains a fairly long run of text to test soft-wrap line-number alignment: Markdown is a lightweight markup language that allows people to write documents in a plain-text format that is easy to read and write, and then convert them into valid XHTML (or HTML) documents. The language incorporates many conventions already familiar from plain-text email markup.

This is the first paragraph, ending with two spaces and a line break.  
This is the second paragraph (hard line break).

This is a new paragraph containing a [reference-style link][ref-link] whose definition is placed at the end of the document.

## 3. Text Emphasis

- *Italic text* and _another italic_
- **Bold text** and __another bold__
- ***Bold italic text***
- ~~Strikethrough text~~
- `Inline code`
- Superscript: X^2^ and H~2~O subscript (supported by some renderers)
- ==Highlighted text== (supported by some renderers)
- **Bold text nesting _italic_ and `code`**
- Emoji test: 😀 🚀 ✅ ❌ 👍 🎉 📝 🔗 🖼️ ⚠️

## 4. Lists

### 4.1 Unordered Lists

- List item one
- List item two
  - Nested list item A
  - Nested list item B
    - Third-level nesting
- List item three

### 4.2 Ordered Lists

1. Step one: open the file
2. Step two: edit the content
   1. Indented ordered sub-item
   2. Second sub-item
3. Step three: save the document

### 4.3 Task Lists

- [x] Completed task
- [x] Markdown rendering supported
- [ ] Incomplete task
- [ ] PDF export support
  - [ ] Nested task one
  - [x] Nested task two

### 4.4 Definition Lists (supported by some renderers)

Term one
: Explanatory content for term one.

Term two
: Explanatory content for term two, which may contain inline formatting such as **bold**.

## 5. Blockquotes

> This is a quoted passage. A quote can contain multiple sentences and is used to test the indentation, border, and text color of blockquote blocks.
>
> The second paragraph inside the quote. A nested quote follows:
>
> > This is a nested quote (second level).
> >
> > > Third-level nested quote.

Other block-level elements can also be placed inside quotes:

> ### Heading inside a quote
>
> - List item one inside a quote
> - List item two inside a quote
>
> Code inside a quote: `const x = 42`

## 6. Code

### 6.1 Inline Code

Run `npm run dev` in the terminal to start the dev server; example path: `C:\Users\admin\Documents\test.md`.

### 6.2 Fenced Code Block (No Language)

```
plain text block
Second line
    Third line with preserved indentation
```

### 6.3 JavaScript

```javascript
// Fibonacci sequence
function fib(n) {
  if (n <= 1) return n
  const dp = [0, 1]
  for (let i = 2; i <= n; i++) {
    dp[i] = dp[i - 1] + dp[i - 2]
  }
  return dp[n]
}

console.log(fib(10)) // 55
```

### 6.4 TypeScript

```typescript
interface User {
  id: number
  name: string
  email?: string
}

const greet = (user: User): string => {
  return `Hello, ${user.name}!`
}
```

### 6.5 Go

```go
package main

import "fmt"

func main() {
	messages := make(chan string, 2)
	messages <- "XinText"
	messages <- "Markdown"
	fmt.Println(<-messages, <-messages)
}
```

### 6.6 Python

```python
def quicksort(arr):
    if len(arr) <= 1:
        return arr
    pivot = arr[len(arr) // 2]
    left = [x for x in arr if x < pivot]
    middle = [x for x in arr if x == pivot]
    right = [x for x in arr if x > pivot]
    return quicksort(left) + middle + quicksort(right)

print(quicksort([3, 6, 1, 8, 2, 9, 4]))
```

### 6.7 JSON / Bash / SQL / HTML

```json
{
  "name": "XinText",
  "version": "1.0.0",
  "features": ["editor", "preview", "export"],
  "active": true
}
```

```bash
#!/bin/bash
for f in *.md; do
  echo "Converting $f"
  pandoc "$f" -o "${f%.md}.html"
done
```

```sql
SELECT u.name, COUNT(p.id) AS post_count
FROM users u
LEFT JOIN posts p ON p.user_id = u.id
WHERE u.created_at > '2026-01-01'
GROUP BY u.name
ORDER BY post_count DESC;
```

```html
<div class="card">
  <h3>Title</h3>
  <p>Hello <strong>world</strong></p>
</div>
```

## 7. Horizontal Rules

Content above.

---

Content below (horizontal rule one).

***

Horizontal rule two.

___

Horizontal rule three.

## 8. Links

- Plain link: [XinText Website](https://example.com/XinText)
- Link with title: [Hover to see the title](https://example.com "Link title tooltip")
- Autolink: <https://www.example.com>
- Email link: <test@example.com>
- Reference-style link: [reference link][ref-link]
- Links with formatting: [**bold link**](https://example.com) and [`code link`](https://example.com)

## 9. Images

![Demo image alt text](https://example.com/images/demo.png "Image title")

Reference-style image:

![Reference image][ref-img]

> Note: In actual use, replace these with local paths or clipboard-pasted images to test absolute and relative path rendering.

## 10. Tables

### 10.1 Basic Table

| Left aligned | Center aligned | Right aligned |
| :--- | :---: | ---: |
| Cell 1 | Cell 2 | 100 |
| Longer cell content test | **bold** | 200 |
| `code` | [link](https://example.com) | 300 |

### 10.2 Compact Syntax

Name|Age|City
-|-|-
Alice|28|Beijing
Bob|35|Shanghai

### 10.3 Table Without Alignment

| Column A | Column B | Column C |
| --- | --- | --- |
| Data | Data | Data |

## 11. Math Formulas (supported by some renderers)

Inline formulas: mass–energy equivalence $E = mc^2$, summation $\sum_{i=1}^{n} i = \frac{n(n+1)}{2}$.

Block formulas:

$$
\frac{\partial f}{\partial x} = 2x + y
$$

$$
\begin{cases}
a_1 x + b_1 y = c_1 \\
a_2 x + b_2 y = c_2
\end{cases}
$$

## 12. Footnotes

Here is a footnote reference[^1], and another one[^longnote]. Footnotes are used to verify superscript markers and jump-to-note rendering at the end of the document.

## 13. Escaped Characters

\* This is not italic \*　　\# This is not a heading　　\` This is not code

Special characters shown verbatim: < > & " ' as well as backslash \\ and pipe \|.

## 14. Inline HTML

Press <kbd>Ctrl</kbd> + <kbd>F</kbd> to open find, <mark>highlighted tag</mark>, <small>small text</small>, <del>deleted</del>, <ins>inserted underline</ins>.

<details>
<summary>Click to expand details</summary>

Content inside the collapsible area, supporting **Markdown** and `code`.

- List item one
- List item two

</details>

## 15. Complex Nesting of Blockquotes, Lists, and Code

1. Explanatory text for step one
   > Note: be especially careful here; this quote sits inside an ordered list.

   ```javascript
   const step = 1
   ```
2. Explanatory text for step two
   - Sub-item A
   - Sub-item B

## 16. Blank Lines and Long-Text Tests (Line-Number Alignment)

Above are blank lines.


Below is a paragraph after two consecutive blank lines. The following line is an extra-long unspaced English string used to test soft wrapping and mirrored line numbers: https://example.com/a/very/long/url/path/that/should/wrap/inside/the/textarea/without/causing/horizontal/scrollbar/issue?foo=bar&baz=qux&lorem=ipsum-dolor-sit-amet-consectetur-adipiscing-elit

Long Chinese line test: in source-code editing mode, text should soft-wrap automatically when it reaches the right edge of the textarea, and the transparent mirror text in the line-number gutter on the left must wrap at exactly the same positions; after wrapping, subsequent line numbers must remain aligned line by line with the original text, with no line-number overlap or misalignment allowed.

## 17. Find Function Test (Ctrl+F)

Keyword test: XinText XinText XinText — the same word appears three times in a row to verify previous/next cyclic navigation.

Chinese keywords: 编辑器, 编辑器, 编辑器, and mixed Markdown编辑器Markdown编辑器 (used to verify Chinese keyword search).

Numbers and symbols: 2026-09-22, version v1.0.0, price ¥99.00, temperature 36.5℃.

## 18. Directory Structure Tree (Plain Text)

```text
XinText/
├── build/
│   ├── bin/
│   │   └── XinText.exe
│   └── windows/
├── frontend/
│   ├── src/
│   │   ├── components/
│   │   ├── composables/
│   │   └── store/
│   └── vite.config.ts
├── internal/
│   └── service/
└── main.go
```

## 19. Ending

End of document. This last paragraph tests whether the source/preview scroll linkage ratio is accurate when scrolling to the bottom.

[^1]: The detailed content of the first footnote.
[^longnote]: A longer footnote that may contain multiple sentences or even `inline code`, used to test the layout of the footnote area.

[ref-link]: https://example.com/reference "Reference-style link title"
[ref-img]: https://example.com/images/ref.png "Reference-style image title"
