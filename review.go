package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// handleReviewCommand launches an interactive review of all goals
func handleReviewCommand() {
	config, client, ok := loadClient(os.Stderr)
	if !ok {
		os.Exit(1)
	}

	// Fetch just the goal list (one request) so the TUI opens immediately. Each
	// goal's datapoints and road are loaded lazily on demand as the user views
	// it (see fetchGoalDetailsCmd), instead of fetching every goal up front —
	// which took ~50s for accounts with many goals.
	goals, err := client.FetchGoals(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to fetch goals: %s\n", redactError(err))
		os.Exit(1)
	}

	if len(goals) == 0 {
		fmt.Println("No goals found.")
		return
	}

	// Sort goals alphabetically by slug as specified
	SortGoalsBySlug(goals)

	// Long-lived context cancelled when the TUI exits, so in-flight lazy detail
	// fetches don't outlive the program (per the client.go context contract).
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Launch the interactive review TUI
	model := initialReviewModel(goals, config)
	model.client = client // use the client built above; the constructor's default is discarded
	model.ctx = ctx
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", redactError(err))
		os.Exit(1)
	}
}

// reviewModel holds the state for the review command
type reviewModel struct {
	goals    []Goal
	details  map[string]*Goal    // lazily-fetched full goals (datapoints, road, …) keyed by slug
	inFlight map[string]struct{} // slugs with a detail fetch currently in flight (dedup)
	loading  bool                // a detail fetch for the current goal is in flight
	ctx      context.Context     // cancelled when the TUI exits; cancels in-flight fetches
	client   Client              // Beeminder API seam; injected so tests can drive detail fetches with a fake
	config   *Config             // credentials for browser-URL/detail rendering (not API calls — those go through client)
	current  int                 // current goal index
	width    int                 // terminal width
	height   int                 // terminal height
	err      string              // error message to display
	viewport viewport.Model      // scrollable pane for the goal content (keeps tall goals reachable on short terminals)
	ready    bool                // viewport has been sized by a WindowSizeMsg
}

// initialReviewModel creates a new review model. The first goal's details fetch
// is dispatched by Init; because Init can't persist model state (it returns only
// a Cmd), the constructor pre-marks that goal as in-flight and loading here.
//
// The client defaults to the real HTTP client for config; handleReviewCommand
// and tests can override m.client to inject a fake before the TUI runs.
func initialReviewModel(goals []Goal, config *Config) reviewModel {
	m := reviewModel{
		goals:    goals,
		details:  make(map[string]*Goal),
		inFlight: make(map[string]struct{}),
		ctx:      context.Background(), // overridden with a cancellable ctx by handleReviewCommand
		client:   NewHTTPClient(config),
		config:   config,
		current:  0,
		loading:  len(goals) > 0,
	}
	if len(goals) > 0 {
		m.inFlight[goals[0].Slug] = struct{}{}
	}
	return m
}

// goalDetailsMsg carries the result of a lazy per-goal details fetch.
type goalDetailsMsg struct {
	slug string
	goal *Goal
	err  error
}

// fetchGoalDetailsCmd fetches one goal's full details (datapoints + road) in the
// background so the TUI opens immediately and navigation stays responsive. The
// context lets the fetch be cancelled when the user quits. The fetch goes through
// the injected Client seam so the review TUI is testable with a fake, like every
// other command.
func fetchGoalDetailsCmd(ctx context.Context, client Client, slug string) tea.Cmd {
	return func() tea.Msg {
		goal, err := client.FetchGoalWithDatapoints(ctx, slug)
		return goalDetailsMsg{slug: slug, goal: goal, err: err}
	}
}

// ensureDetails returns a command to fetch the current goal's details if they
// aren't already cached or in flight, updating the loading flag accordingly.
// Deduping on inFlight stops rapid navigation (away and back before a fetch
// resolves) from firing a second request for the same goal.
func (m *reviewModel) ensureDetails() tea.Cmd {
	if len(m.goals) == 0 {
		m.loading = false
		return nil
	}
	slug := m.goals[m.current].Slug
	if _, ok := m.details[slug]; ok {
		m.loading = false
		return nil
	}
	m.loading = true
	if _, ok := m.inFlight[slug]; ok {
		return nil // already fetching this goal; just keep showing the spinner
	}
	m.inFlight[slug] = struct{}{}
	return fetchGoalDetailsCmd(m.ctx, m.client, slug)
}

func (m reviewModel) Init() tea.Cmd {
	// The constructor already marked goals[0] in-flight; just dispatch its fetch.
	if len(m.goals) == 0 {
		return nil
	}
	return fetchGoalDetailsCmd(m.ctx, m.client, m.goals[0].Slug)
}

func (m reviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Reserve the bottom rows for the pinned help bar; the rest is the
		// scrollable content pane so tall goals stay reachable on short
		// terminals (the content used to overflow the alt screen and the top
		// scrolled off with no way to reach it).
		helpHeight := lipgloss.Height(m.helpView())
		contentHeight := max(1, msg.Height-helpHeight)
		if !m.ready {
			m.viewport = viewport.New(msg.Width, contentHeight)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = contentHeight
		}
		m.refreshContent()
		return m, nil

	case goalDetailsMsg:
		// This fetch is no longer in flight.
		delete(m.inFlight, msg.slug)
		// Cache the result regardless of which goal is now current (the user
		// may have navigated on). Only touch loading/err for the current goal.
		isCurrent := len(m.goals) > 0 && msg.slug == m.goals[m.current].Slug
		if msg.err != nil {
			if isCurrent {
				m.loading = false
				m.err = fmt.Sprintf("Failed to load goal details: %s", redactError(msg.err))
				// Re-flow so the error replaces the "Loading…" line in the pane;
				// both are rendered inside contentView, which the viewport only
				// re-reads on refresh.
				m.refreshContent()
			}
			return m, nil
		}
		m.details[msg.slug] = msg.goal
		if isCurrent {
			m.loading = false
			m.err = ""
			// Datapoints/chart just arrived for the goal on screen; re-flow the
			// content pane so they appear (keeping the current scroll position).
			m.refreshContent()
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit

		case "right", "l", "n", "j":
			// Next goal
			if m.current < len(m.goals)-1 {
				m.current++
			}
			m.err = ""
			cmd := m.ensureDetails()
			// New goal: re-flow and jump back to the top of the pane.
			m.refreshContent()
			m.viewport.GotoTop()
			return m, cmd

		case "left", "h", "p", "k":
			// Previous goal
			if m.current > 0 {
				m.current--
			}
			m.err = ""
			cmd := m.ensureDetails()
			m.refreshContent()
			m.viewport.GotoTop()
			return m, cmd

		case "o", "enter":
			// Open current goal in browser
			if m.current < len(m.goals) {
				goal := m.goals[m.current]
				if err := openBrowser(m.config, goal.Slug); err != nil {
					m.err = fmt.Sprintf("Failed to open browser: %v", err)
				} else {
					m.err = "" // Clear any previous error
				}
				m.refreshContent()
			}
			return m, nil
		}
	}

	// Anything else (↑/↓, PgUp/PgDn, Home/End, mouse wheel, …) scrolls the
	// content pane. Goal-navigation and action keys returned above, so they
	// never reach the viewport.
	if !m.ready {
		return m, nil
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// refreshContent re-renders the goal content into the scroll pane. No-op until
// the viewport has been sized (e.g. before the first WindowSizeMsg, or in tests
// that call View directly), where View falls back to rendering content inline.
func (m *reviewModel) refreshContent() {
	if m.ready {
		m.viewport.SetContent(m.contentView())
	}
}

func (m reviewModel) View() string {
	if len(m.goals) == 0 {
		return "No goals to review.\n\nPress q to quit."
	}

	// Once sized by a WindowSizeMsg, render the content through the scroll pane
	// so tall goals stay reachable. Before that (and in tests that call View
	// directly) fall back to rendering the full content inline, unscrolled.
	if m.ready {
		return m.viewport.View() + "\n" + m.helpView()
	}
	// contentView can end with a trailing newline (e.g. the loading/error line);
	// trim it so the inline fallback doesn't stack a blank row before the help
	// bar, matching the viewport path (viewport.View has no trailing newline).
	return strings.TrimRight(m.contentView(), "\n") + "\n" + m.helpView()
}

// contentView renders the scrollable portion of the review screen: the goal
// title, details, chart, loading indicator, and any error. The help bar is
// rendered separately (helpView) and pinned below the scroll pane.
func (m reviewModel) contentView() string {
	if len(m.goals) == 0 {
		return ""
	}

	// Start from the bulk summary goal, then merge in the detail-only fields;
	// see Goal.hydrateFrom for which fields and why it merges rather than replaces.
	goal := m.goals[m.current]
	if d, ok := m.details[goal.Slug]; ok {
		goal.hydrateFrom(d)
	}

	// Create the goal details view
	var view string

	// Title section with counter and status indicator
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")).
		Padding(0, 1)

	counterStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Padding(0, 1)

	// Colored status indicator. Uses bright-palette variants (9/214/12/10/241)
	// rather than the main urgency colours so the dot stands out next to the
	// title text, which already uses the main palette.
	var statusColor lipgloss.Color
	statusSymbol := "●"
	switch UrgencyFor(goal.Safebuf) {
	case UrgencyOverdue:
		statusColor = lipgloss.Color("9")
	case UrgencyDueToday:
		statusColor = lipgloss.Color("214")
	case UrgencyDueTomorrow:
		statusColor = lipgloss.Color("12")
	case UrgencyThisWeek:
		statusColor = lipgloss.Color("10")
	default:
		statusColor = lipgloss.Color("241")
	}

	statusStyle := lipgloss.NewStyle().
		Foreground(statusColor).
		Padding(0, 1, 0, 0)

	view += statusStyle.Render(statusSymbol) + titleStyle.Render(fmt.Sprintf("Goal: %s", goal.Slug)) + "\n"
	view += counterStyle.Render(fmt.Sprintf("Goal %d of %d", m.current+1, len(m.goals))) + "\n\n"

	// Goal details section
	detailStyle := lipgloss.NewStyle().
		Padding(0, 2)

	details := formatGoalDetails(&goal, m.config, time.Now())

	view += detailStyle.Render(details) + "\n"

	// Progress chart (datapoints vs. bright red line). Empty when the goal has
	// no datapoints or none inside the charted window.
	if chart := renderGoalChart(goal, m.width); chart != "" {
		view += chart
	}

	// Loading indicator while this goal's datapoints/chart are being fetched.
	if m.loading {
		loadingStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Padding(0, 2)
		view += loadingStyle.Render("Loading datapoints…") + "\n"
	}

	// Error message section (if any). Errors are free-form (e.g. a full API URL
	// in a fetch failure), so wrap to the terminal width instead of letting the
	// line overflow and get cut off. Width includes the horizontal padding.
	if m.err != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Padding(0, 2)
		if m.width > 0 {
			errorStyle = errorStyle.Width(m.width)
		}
		view += errorStyle.Render(fmt.Sprintf("⚠ %s", m.err)) + "\n"
	}

	return view
}

// helpView renders the key hints pinned below the scroll pane. When the content
// overflows the pane, it also shows a scroll position so the user knows there's
// more above or below.
func (m reviewModel) helpView() string {
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Padding(1, 2)

	help := "Navigation: ← → (or h l, or j k, or p n)  |  Scroll: ↑ ↓ PgUp PgDn  |  Open in browser: o or Enter  |  Quit: q or Esc"
	// Reserve the indicator's slot whether or not the percentage is shown, so the
	// help bar keeps a constant width as the user moves between goals that do and
	// don't overflow (a varying width could shift terminal wrapping on narrow
	// widths). %3.0f%% is always 4 chars, so the slot is a fixed "  |  100%" wide.
	if m.ready && (!m.viewport.AtTop() || !m.viewport.AtBottom()) {
		help += fmt.Sprintf("  |  %3.0f%%", m.viewport.ScrollPercent()*100)
	} else {
		help += strings.Repeat(" ", len("  |  100%"))
	}
	return helpStyle.Render(help)
}

// openBrowser opens the goal page in the default browser
func openBrowser(config *Config, goalSlug string) error {
	baseURL := getBaseURL(config)
	goalURL := fmt.Sprintf("%s/%s/%s", baseURL, url.PathEscape(config.Username), url.PathEscape(goalSlug))

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", goalURL)
	case "linux":
		cmd = exec.Command("xdg-open", goalURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", goalURL)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}
