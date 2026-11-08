package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// ? opens a panel of the page's keys and the ones that work everywhere,
// and esc or ? closes it; another key changes nothing while it is open.
func TestHelpPanelListsThePagesKeys(t *testing.T) {
	home(t)
	m := newModel()
	m.w, m.h = 120, 40
	m = press(t, m, "?")
	if !m.help {
		t.Fatal("? didn't open the help panel")
	}
	for _, want := range []string{"keys · agents", "pick an agent", "pick a field", "everywhere", "go to a page", "ctrl+c"} {
		if !strings.Contains(m.View(), want) {
			t.Errorf("the agents help doesn't name %q:\n%s", want, m.View())
		}
	}
	// every page has its own title and its own keys
	for _, c := range []struct {
		p    page
		want string
	}{
		{pageProviders, "fetch models"},
		{pageGroups, "bring a removed one back"},
		{pageUsage, "daily check-in"},
		{pageSessions, "narrow to a folder"},
		{pageLibrary, "update a skill"},
	} {
		m.page, m.help = c.p, true
		if !strings.Contains(m.View(), c.want) {
			t.Errorf("the %s help doesn't name %q:\n%s", pageNames[c.p], c.want, m.View())
		}
	}
	m.page = pageAgents
	m = press(t, m, "esc")
	if m.help {
		t.Fatal("esc didn't close the help panel")
	}
	// a key meant for the page under it does nothing while the panel is open
	m = press(t, m, "?", "q")
	if !m.help {
		t.Fatal("q closed the help panel instead of nothing")
	}
	m = press(t, m, "?")
	if m.help {
		t.Fatal("? didn't close the help panel")
	}
}

// The help panel fits the terminal: no line is wider than the screen at
// 80 or 120 columns.
func TestHelpPanelFitsTheScreen(t *testing.T) {
	home(t)
	for _, size := range [][2]int{{80, 24}, {120, 40}} {
		for p := page(0); p < page(len(pageNames)); p++ {
			m := newModel()
			m.w, m.h = size[0], size[1]
			m.page, m.help = p, true
			for _, line := range strings.Split(m.View(), "\n") {
				if w := lipgloss.Width(line); w > size[0] {
					t.Errorf("%dx%d %s: a line is %d wide: %q", size[0], size[1], pageNames[p], w, line)
				}
			}
		}
	}
}

// The footer at 80 columns keeps the hints read most, whole, and ends in
// "? help": the long tail is left out, never cut in the middle of a word.
func TestFooterHintsFitEightyColumns(t *testing.T) {
	home(t)
	for _, c := range []struct {
		p      page
		must   []string
		absent string
	}{
		{pageAgents, []string{"↑↓ agent", "←→ field", "↵ change", "q quit"}, "1–6 pages"},
		{pageProviders, []string{"↑↓ provider", "↵ models", "a add", "q quit"}, "balances"},
		{pageGroups, []string{"↑↓ group", "↵ open", "n new", "q quit"}, "bring back"},
		{pageUsage, []string{"←→ period", "u used / left", "c daily check-in", "q quit"}, "today · 7 days"},
		{pageSessions, []string{"↑↓ session", "↵ resume", "q quit"}, "tokens / cost"},
		{pageLibrary, []string{"↑↓ item", "↵ agents", "a add", "q quit"}, "bring in"},
	} {
		m := newModel()
		m.w, m.h = 80, 24
		m.page = c.p
		m.reload()
		lines := strings.Split(strings.TrimRight(m.View(), "\n"), "\n")
		line := lines[len(lines)-1]
		if w := lipgloss.Width(line); w > 80 {
			t.Errorf("%s footer is %d wide: %q", pageNames[c.p], w, line)
		}
		if !strings.HasSuffix(line, "? help") {
			t.Errorf("%s footer doesn't end in ? help: %q", pageNames[c.p], line)
		}
		for _, want := range c.must {
			if !strings.Contains(line, want) {
				t.Errorf("%s footer doesn't keep %q: %q", pageNames[c.p], want, line)
			}
		}
		if strings.Contains(line, c.absent) {
			t.Errorf("%s footer kept the low-priority %q: %q", pageNames[c.p], c.absent, line)
		}
	}
	// 120 columns has room for the tail the 80-column line left out
	m := newModel()
	m.w, m.h = 120, 40
	if line := strings.Split(strings.TrimRight(m.View(), "\n"), "\n"); !strings.Contains(line[len(line)-1], "1–6 pages") {
		t.Errorf("the agents footer at 120 drops the pages: %q", line[len(line)-1])
	}
}
