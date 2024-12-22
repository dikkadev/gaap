package selector

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dikkadev/gaap/pkg/github"
)

type RepoItem struct {
	repo github.Repository
}

func (i RepoItem) Title() string {
	return i.repo.FullName
}

func (i RepoItem) Description() string {
	desc := i.repo.Description
	prefix := fmt.Sprintf("⭐ %d | ", i.repo.Stars)
	maxLen := 100 - len(prefix)
	if len(desc) > maxLen {
		desc = desc[:maxLen-3] + "..."
	}
	return prefix + desc
}

func (i RepoItem) FilterValue() string {
	return i.repo.FullName
}

type model struct {
	list       list.Model
	selected   *github.Repository
	err        error
	quitting   bool
	totalCount int
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			if i, ok := m.list.SelectedItem().(RepoItem); ok {
				m.selected = &i.repo
				return m, tea.Quit
			}
		case "ctrl+n":
			m.list.CursorDown()
		case "ctrl+p":
			m.list.CursorUp()
		case "pgdown", "ctrl+d":
			m.list.NextPage()
		case "pgup", "ctrl+u":
			m.list.PrevPage()
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\nPress any key to exit\n", m.err)
	}

	if m.quitting {
		return ""
	}

	help := "\nNavigate: ↑/↓ • Page: PgUp/PgDn • Filter: / • Select: Enter • Quit: Esc/q\n"
	return m.list.View() + help
}

// searchRepositories performs the repository search without any UI interaction
func searchRepositories(ctx context.Context, ghClient github.Client, input string) (*github.SearchResult, error) {
	var result *github.SearchResult
	var err error

	// Try exact match first if input contains a slash
	if strings.Contains(input, "/") {
		parts := strings.Split(input, "/")
		if len(parts) == 2 {
			// Try exact match first
			result, err = ghClient.SearchRepositories(ctx, fmt.Sprintf("repo:%s/%s", parts[0], parts[1]))
			if err != nil {
				return nil, fmt.Errorf("failed to search for exact match: %w", err)
			}
			if result.TotalCount == 1 {
				return result, nil
			}

			// If not exact match, try fuzzy search with both parts
			result, err = ghClient.SearchRepositories(ctx, fmt.Sprintf("user:%s %s in:name", parts[0], parts[1]))
			if err != nil {
				return nil, fmt.Errorf("failed to search with user and name: %w", err)
			}
		}
	}

	// If no results yet, try searching by repo name
	if result == nil || result.TotalCount == 0 {
		result, err = ghClient.SearchRepositoriesByName(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to search by name: %w", err)
		}
	}

	// If still no results, try searching by user
	if result.TotalCount == 0 {
		result, err = ghClient.SearchRepositoriesByUser(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to search by user: %w", err)
		}
	}

	if result.TotalCount == 0 {
		return nil, fmt.Errorf("no repositories found matching '%s'", input)
	}

	return result, nil
}

// SelectRepository presents an interactive UI for selecting a repository from search results
func SelectRepository(ctx context.Context, ghClient github.Client, input string) (*github.Repository, error) {
	result, err := searchRepositories(ctx, ghClient, input)
	if err != nil {
		return nil, err
	}

	// For exact matches with a single result, return immediately
	if strings.Contains(input, "/") && result.TotalCount == 1 {
		return &result.Items[0], nil
	}

	// Convert results to list items
	items := make([]list.Item, len(result.Items))
	for i, repo := range result.Items {
		items[i] = RepoItem{repo: repo}
	}

	// Create the list model with appropriate size
	width := 80
	height := min(20, len(items)+5) // 5 lines for header, help, etc.
	l := list.New(items, list.NewDefaultDelegate(), width, height)
	l.Title = fmt.Sprintf("Select a repository (found %d)", result.TotalCount)
	l.SetShowHelp(true)
	l.SetFilteringEnabled(true)
	l.SetShowStatusBar(true)
	l.SetShowTitle(true)
	l.KeyMap.Quit.SetEnabled(true)
	l.KeyMap.ForceQuit.SetEnabled(true)
	l.SetShowFilter(true)
	l.SetFilteringEnabled(true)

	// Create and run the model
	m := model{
		list:       l,
		totalCount: result.TotalCount,
	}

	prog := tea.NewProgram(m)
	finalModel, err := prog.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to run UI: %w", err)
	}

	if m, ok := finalModel.(model); ok && m.selected != nil {
		return m.selected, nil
	}

	return nil, fmt.Errorf("no repository selected")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
