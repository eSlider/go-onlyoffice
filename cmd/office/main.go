// Command office is a terminal UI for OnlyOffice Workspace.
//
// Three-pane layout: module tree (left), selectable list (center),
// markdown preview (right). Uses github.com/eslider/go-onlyoffice for API calls.
//
// Build & install:
//
//	go install github.com/eslider/go-onlyoffice/cmd/office@latest
package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/eslider/go-onlyoffice/cmd/internal/bootstrap"
	"github.com/eslider/go-onlyoffice/cmd/office/ui"
)

func main() {
	client, err := bootstrap.NewClient(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	m := ui.NewModel(client)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
