# Markdown 全特性测试文档

> 本文档用于测试 XinText 编辑器对 Markdown 各种语法的渲染效果，涵盖标题、段落、强调、列表、引用、代码、链接、图片、表格、任务列表、脚注、数学公式、Emoji、HTML 内联等特性。中英文混排测试 Mixed English and 中文 Content，数字 1234567890。

## 1. 标题

# 一级标题 H1

## 二级标题 H2

### 三级标题 H3

#### 四级标题 H4

##### 五级标题 H5

###### 六级标题 H6

## 2. 段落与换行

这是一个普通段落。段落内包含较长的文字内容用于测试软折行行号对齐：Markdown 是一种轻量级标记语言，它允许人们使用易读易写的纯文本格式编写文档，然后转换成有效的 XHTML（或者 HTML）文档。这种语言吸收了很多在电子邮件中已有的纯文本标记的特性。

这是第一段，末尾有两个空格加换行。  
这是第二段（硬换行）。

这是新的一段，中间包含一个[链接示例][ref-link]引用式链接定义在文末。

## 3. 文本强调

- *斜体文本* 与 _另一种斜体_
- **粗体文本** 与 __另一种粗体__
- ***粗斜体文本***
- ~~删除线文本~~
- `行内代码 inline code`
- 上标：X^2^ 与 H~2~O 下标（部分渲染器支持）
- ==高亮文本==（部分渲染器支持）
- **粗体中嵌套 _斜体_ 与 `代码`**
- Emoji 测试：😀 🚀 ✅ ❌ 👍 🎉 📝 🔗 🖼️ ⚠️

## 4. 列表

### 4.1 无序列表

- 列表项一
- 列表项二
  - 嵌套列表项 A
  - 嵌套列表项 B
    - 第三层嵌套
- 列表项三

### 4.2 有序列表

1. 第一步：打开文件
2. 第二步：编辑内容
   1. 缩进的有序子项
   2. 第二个子项
3. 第三步：保存文档

### 4.3 任务列表

- [x] 已完成的任务
- [x] 支持 Markdown 渲染
- [ ] 未完成的任务
- [ ] 支持导出 PDF
  - [ ] 嵌套任务一
  - [x] 嵌套任务二

### 4.4 定义列表（部分渲染器支持）

术语一
: 术语一的解释说明内容。

术语二
: 术语二的解释说明内容，可以包含**粗体**等行内格式。

## 5. 引用

> 这是一段引用文字。引用可以包含多个句子，用于测试引用块的缩进、边框与文字颜色表现。
>
> 引用中的第二段。嵌套引用如下：
>
> > 这是嵌套引用（第二层）。
> >
> > > 第三层嵌套引用。

引用中也可以放其他块级元素：

> ### 引用中的标题
>
> - 引用中的列表项一
> - 引用中的列表项二
>
> 引用中的代码：`const x = 42`

## 6. 代码

### 6.1 行内代码

在命令行执行 `npm run dev` 启动开发服务器，路径示例 `C:\Users\admin\Documents\test.md`。

### 6.2 围栏代码块（无语言）

```
plain text block
第二行
    保留缩进的第三行
```

### 6.3 JavaScript

```javascript
// 斐波那契数列
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

## 7. 分割线

上面内容。

---

下面内容（分割线一）。

***

分割线二。

___

分割线三。

## 8. 链接

- 普通链接：[XinText 官网](https://example.com/XinText)
- 带标题链接：[悬停查看标题](https://example.com "链接标题提示")
- 自动链接：<https://www.example.com>
- 邮箱链接：<test@example.com>
- 引用式链接：[参考链接][ref-link]
- 链接中含格式：[**粗体链接**](https://example.com) 与 [`代码链接`](https://example.com)

## 9. 图片

![示例图片 Alt 文本](https://example.com/images/demo.png "图片标题")

引用式图片：

![引用图片][ref-img]

> 注：实际使用时可替换为本地路径或剪贴板粘贴的图片，测试绝对路径与相对路径渲染。

## 10. 表格

### 10.1 基础表格

| 左对齐 | 居中对齐 | 右对齐 |
| :--- | :---: | ---: |
| 单元格 1 | 单元格 2 | 100 |
| 较长的单元格内容测试 | **粗体** | 200 |
| `code` | [链接](https://example.com) | 300 |

### 10.2 紧凑写法

姓名|年龄|城市
-|-|-
张三|28|北京
李四|35|上海

### 10.3 无对齐表格

| 列 A | 列 B | 列 C |
| --- | --- | --- |
| 数据 | 数据 | 数据 |

## 11. 数学公式（部分渲染器支持）

行内公式：质能方程 $E = mc^2$，求和 $\sum_{i=1}^{n} i = \frac{n(n+1)}{2}$。

块级公式：

$$
\frac{\partial f}{\partial x} = 2x + y
$$

$$
\begin{cases}
a_1 x + b_1 y = c_1 \\
a_2 x + b_2 y = c_2
\end{cases}
$$

## 12. 脚注

这里有一个脚注引用[^1]，再来一个[^longnote]。脚注测试用于验证上角标与文末注释的跳转渲染。

## 13. 转义字符

\* 不是斜体 \*　　\# 不是标题　　\` 不是代码

特殊符号原样显示：< > & " ' 以及反斜杠 \\ 和管道 \|。

## 14. 内联 HTML

<kbd>Ctrl</kbd> + <kbd>F</kbd> 打开查找，<mark>高亮标签</mark>，<small>小字</small>，<del>删除</del>，<ins>下划线插入</ins>。

<details>
<summary>点击展开详情</summary>

折叠区域内的内容，支持 **Markdown** 与 `代码`。

- 列表项一
- 列表项二

</details>

## 15. 引用块、列表与代码的复杂嵌套

1. 第一步说明文字
   > 注意：此处需要特别小心，引用位于有序列表内部。

   ```javascript
   const step = 1
   ```
2. 第二步说明文字
   - 子项 A
   - 子项 B

## 16. 空行与长文本测试（行号对齐）

上面是空行。


下面是连续两个空行之后的段落。下面这一行是没有任何空格的超长英文串，用于测试软折行与镜像行号：https://example.com/a/very/long/url/path/that/should/wrap/inside/the/textarea/without/causing/horizontal/scrollbar/issue?foo=bar&baz=qux&lorem=ipsum-dolor-sit-amet-consectetur-adipiscing-elit

中文长行测试：编辑器源码模式下，文字应当在到达 textarea 右边界时自动软折行，左侧行号槽中的透明镜像文本必须以完全相同的位置折行，折行之后的下一行行号仍然与原文逐行对齐，不允许出现行号重叠或错位现象。

## 17. 查找功能测试（Ctrl+F）

关键词测试：XinText XinText XinText，连续出现三次相同单词，用于验证上一个/下一个循环定位。

中文关键词：编辑器、编辑器、编辑器，以及混排 Markdown编辑器Markdown编辑器。

数字与符号：2026-09-22，版本 v1.0.0，价格 ¥99.00，温度 36.5℃。

## 18. 目录结构树（纯文本）

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

## 19. 结尾

文档结束。最后一段用于测试滚动到底部时源码与预览的联动比例是否准确。

[^1]: 这是第一个脚注的详细内容。
[^longnote]: 较长的脚注内容，可以包含多个句子甚至 `行内代码`，用于测试脚注区域的排版。

[ref-link]: https://example.com/reference "引用式链接标题"
[ref-img]: https://example.com/images/ref.png "引用式图片标题"
