package preview

import (
	"github.com/charmbracelet/glamour"
)

// RenderMarkdown renders GitHub-flavoured markdown for terminal display.
func RenderMarkdown(md string, width int) (string, error) {
	if md == "" {
		return "", nil
	}
	if width < 20 {
		width = 20
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return "", err
	}
	return r.Render(md)
}
