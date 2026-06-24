package preview

import (
	md "github.com/JohannesKaufmann/html-to-markdown/v2"
)

// HTMLToMarkdown converts Document Server HTML output to markdown for TUI preview.
func HTMLToMarkdown(html string) (string, error) {
	return md.ConvertString(html)
}
