package timemachine

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tinyzimmer/btrsync/cmd/btrsync/cmd/config"
)

type Model struct {
	// Config containing volumes and subvolumes
	conf *config.Config
	// Current volume
	volumeCursor int
	// Current subvolume
	subvolumeCursor int
}

func Run(conf *config.Config) error {
	p := tea.NewProgram(Model{conf: conf}, tea.WithMouseAllMotion())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}

func (m Model) Init() tea.Cmd { return tea.ClearScreen }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Is it a key press?
	case tea.KeyMsg:

		// Cool, what was the actual key pressed?
		switch msg.String() {

		// These keys should exit the program.
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.volumeCursor > 0 {
				m.volumeCursor--
			}

		// The "down" and "j" keys move the cursor down
		case "down", "j":
			if m.volumeCursor < len(m.conf.Volumes)-1 {
				m.volumeCursor++
			}

		}
	}

	// Return the updated model to the Bubble Tea runtime for processing.
	// Note that we're not returning a command.
	return m, nil
}

func (m Model) View() string {
	// The header
	s := "Select a volume\n\n"

	for i, vol := range m.conf.Volumes {

		// Is the cursor pointing at this choice?
		cursor := " " // no cursor
		if m.volumeCursor == i {
			cursor = ">" // cursor!
		}

		// Render the row
		s += fmt.Sprintf("%s %s\n", cursor, vol.GetName())
	}

	// The footer
	s += "\nPress q to quit.\n"

	// Send the UI for rendering
	return s
}
