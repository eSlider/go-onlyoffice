package ui

import (
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/eslider/go-onlyoffice/cmd/office/model"
	"github.com/mattn/go-runewidth"
	"github.com/muesli/termenv"
)

func TestProjectTableHeaderFitsPaneWidth(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })

	for _, w := range []int{68, 50, 40, 30} {
		tbl := newDataTable()
		tbl.SetSize(w, 10)
		tbl.SetData(model.ListSpec{Subject: model.SubjectProjects}, []model.Item{
			{ID: "1", Title: "Alpha", Kind: model.KindProject, Raw: map[string]any{"status": 0}},
		})
		tbl.SetFocused(true)
		re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
		header := tbl.renderHeader()
		got := runewidth.StringWidth(re.ReplaceAllString(header, ""))
		if got != w {
			t.Fatalf("width=%d header display width=%d", w, got)
		}
		view := tbl.View()
		line0 := strings.Split(strings.TrimSuffix(view, "\n"), "\n")[0]
		gotView := runewidth.StringWidth(re.ReplaceAllString(line0, ""))
		if gotView < w-1 || gotView > w+1 {
			t.Fatalf("width=%d view header width=%d", w, gotView)
		}
	}
}

func TestPadANSIWidthPreservesStyledLine(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })

	line := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")).Render("ID")
	padded := padANSIWidth(line, 10)
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	if runewidth.StringWidth(re.ReplaceAllString(padded, "")) != 10 {
		t.Fatalf("expected padded width 10, got %q", padded)
	}
	if !strings.Contains(padded, "ID") {
		t.Fatalf("padding must not destroy cell text: %q", padded)
	}
}
