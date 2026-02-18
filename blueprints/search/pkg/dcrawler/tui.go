package dcrawler

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- Styles ---

var (
	tuiPrimary   = lipgloss.Color("#1a73e8")
	tuiSuccess   = lipgloss.Color("#1e8e3e")
	tuiError     = lipgloss.Color("#d93025")
	tuiWarning   = lipgloss.Color("#f9ab00")
	tuiDim       = lipgloss.Color("#9aa0a6")

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ffffff")).
			Background(tuiPrimary).
			Padding(0, 1)

	tuiLabelStyle = lipgloss.NewStyle().
			Foreground(tuiDim).
			Width(10)

	tuiValueStyle = lipgloss.NewStyle().
			Bold(true)

	tuiSuccessStyle = lipgloss.NewStyle().Foreground(tuiSuccess)
	tuiErrorStyle   = lipgloss.NewStyle().Foreground(tuiError)
	tuiWarningStyle = lipgloss.NewStyle().Foreground(tuiWarning)
	tuiDimStyle     = lipgloss.NewStyle().Foreground(tuiDim)
	tuiBoldStyle    = lipgloss.NewStyle().Bold(true)
)

// --- Messages ---

type tickMsg time.Time
type phaseMsg string

func tickCmd() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// --- Model ---

type tuiModel struct {
	stats     *Stats
	config    Config
	phase     string // "init", "crawl", "done"
	initLines []string
	width     int
	height    int
	spinner   spinner.Model
	progress  progress.Model
	err       error
	quitting  bool
	cancel    func() // cancel crawl context
}

func newTUIModel(stats *Stats, cfg Config, cancel func()) tuiModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(tuiPrimary)

	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(50),
		progress.WithoutPercentage(),
	)

	return tuiModel{
		stats:    stats,
		config:   cfg,
		phase:    "init",
		spinner:  s,
		progress: p,
		cancel:   cancel,
	}
}

func (m tuiModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, tickCmd())
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			if m.cancel != nil {
				m.cancel()
			}
			m.phase = "done"
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		barWidth := msg.Width - 30
		if barWidth < 20 {
			barWidth = 20
		}
		if barWidth > 80 {
			barWidth = 80
		}
		m.progress = progress.New(
			progress.WithDefaultGradient(),
			progress.WithWidth(barWidth),
			progress.WithoutPercentage(),
		)
		return m, nil

	case tickMsg:
		// Drain init log messages
		if m.stats.initLog != nil {
			for {
				select {
				case msg := <-m.stats.initLog:
					m.initLines = append(m.initLines, msg)
				default:
					goto drained
				}
			}
		drained:
		}

		// Auto-detect crawl phase from stats
		if m.phase == "init" && m.stats.Done() > 0 {
			m.phase = "crawl"
		}
		return m, tickCmd()

	case phaseMsg:
		m.phase = string(msg)
		if m.phase == "done" {
			m.stats.Freeze()
			return m, tea.Quit
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m tuiModel) View() string {
	if m.width == 0 {
		return "" // wait for first WindowSizeMsg
	}

	var sections []string

	// === Header Panel ===
	sections = append(sections, m.viewHeader())

	// === Progress ===
	sections = append(sections, m.viewProgress())

	if m.phase == "init" {
		sections = append(sections, m.viewInit())
	} else {
		// === Stats ===
		sections = append(sections, m.viewStats())

		// === Results ===
		sections = append(sections, m.viewResults())

		// === Rod workers ===
		if m.stats.useRod {
			sections = append(sections, m.viewWorkers())
		}

		// === HTTP + Depth ===
		sections = append(sections, m.viewHTTPDepth())
	}

	// === Footer ===
	sections = append(sections, "")
	if m.phase == "done" {
		sections = append(sections, tuiDimStyle.Render("  Crawl finished. Exiting..."))
	} else {
		sections = append(sections, tuiDimStyle.Render("  Press q to quit"))
	}

	return strings.Join(sections, "\n")
}

// --- View components ---

func (m tuiModel) viewHeader() string {
	domain := NormalizeDomain(m.config.Domain)

	// Mode description
	var mode string
	if m.config.UseLightpanda {
		w := m.config.RodWorkers
		if w <= 0 {
			w = 8
		}
		mode = fmt.Sprintf("lightpanda · %d processes", w)
	} else if m.config.UseRod {
		w := m.config.RodWorkers
		if w <= 0 {
			w = 40
		}
		mode = fmt.Sprintf("headless Chrome · %d pages", w)
	} else {
		h2 := "HTTP/2"
		if m.config.ForceHTTP1 {
			h2 = "HTTP/1.1"
		}
		mode = fmt.Sprintf("%d workers · %d conns · %s", m.config.Workers, m.config.MaxConns, h2)
	}

	// Limits
	maxDepth := "unlimited depth"
	if m.config.MaxDepth > 0 {
		maxDepth = fmt.Sprintf("depth %d", m.config.MaxDepth)
	}
	maxPages := "unlimited pages"
	if m.config.MaxPages > 0 {
		maxPages = fmt.Sprintf("%s pages", fmtInt(m.config.MaxPages))
	}
	if m.config.Continuous {
		maxPages = "continuous"
	}

	title := headerStyle.Render(" Domain Crawler ")
	line1 := fmt.Sprintf("  %s  %s  %s", tuiBoldStyle.Render(domain), tuiDimStyle.Render("·"), mode)
	line2 := fmt.Sprintf("  %s · %s · %s", maxDepth, maxPages, tuiDimStyle.Render(m.config.DomainDir()))

	content := title + "\n" + line1 + "\n" + line2
	return content
}

func (m tuiModel) viewProgress() string {
	succ := m.stats.success.Load()
	done := m.stats.Done()
	elapsed := m.stats.Elapsed()

	if m.config.MaxPages > 0 {
		pct := float64(succ) / float64(m.config.MaxPages)
		if pct > 1 {
			pct = 1
		}
		bar := m.progress.ViewAs(pct)
		return fmt.Sprintf("\n  %s  %s ok / %s target  (%s total)",
			bar, fmtInt64(succ), fmtInt(m.config.MaxPages), fmtInt64(done))
	}

	// Open-ended: pulsing effect
	barWidth := 50
	pos := int(elapsed.Seconds()*2) % (barWidth * 2)
	if pos >= barWidth {
		pos = barWidth*2 - pos - 1
	}
	var bar strings.Builder
	for i := range barWidth {
		if i >= pos-1 && i <= pos+1 {
			bar.WriteString("\u2588")
		} else {
			bar.WriteString("\u2591")
		}
	}
	modeLabel := ""
	if m.config.Continuous {
		modeLabel = " [continuous]"
	}
	return fmt.Sprintf("\n  %s  %s pages%s", bar.String(), fmtInt64(done), modeLabel)
}

func (m tuiModel) viewInit() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("\n  %s Initializing crawler...\n", m.spinner.View()))
	for _, line := range m.initLines {
		b.WriteString(fmt.Sprintf("  %s %s\n", tuiSuccessStyle.Render("\u2713"), line))
	}
	return b.String()
}

func (m tuiModel) viewStats() string {
	succ := m.stats.success.Load()
	bytesTotal := m.stats.bytes.Load()
	done := m.stats.Done()
	inflight := m.stats.inFlight.Load()
	speed := m.stats.Speed()
	elapsed := m.stats.Elapsed()
	bw := m.stats.ByteSpeed()

	// ETA
	eta := "---"
	if m.config.Continuous {
		eta = "continuous"
	} else if m.config.MaxPages > 0 && elapsed.Seconds() > 2 && succ > 0 && speed > 0 {
		successRate := float64(succ) / float64(done)
		effectiveSpeed := speed * successRate
		remaining := int64(m.config.MaxPages) - succ
		if remaining > 0 && effectiveSpeed > 0 {
			etaDur := time.Duration(float64(remaining)/effectiveSpeed) * time.Second
			eta = formatDuration(etaDur)
		} else {
			eta = "done"
		}
	}

	avgPage := int64(0)
	if succ > 0 {
		avgPage = bytesTotal / succ
	}

	var b strings.Builder
	b.WriteString("\n")

	// Speed
	b.WriteString(fmt.Sprintf("  %s %s pages/s  \u2502  Peak %s/s  \u2502  BW %s/s\n",
		tuiLabelStyle.Render("Speed"),
		tuiValueStyle.Render(fmtInt64(int64(speed))),
		fmtInt64(int64(m.stats.peakSpeed)),
		fmtBytes(int64(bw))))

	// Pages
	b.WriteString(fmt.Sprintf("  %s %s done  \u2502  %s ok (%4.1f%%)  \u2502  %s  \u2502  avg %s/page\n",
		tuiLabelStyle.Render("Pages"),
		tuiValueStyle.Render(fmtInt64(done)),
		tuiSuccessStyle.Render(fmtInt64(succ)),
		safePct(succ, done),
		fmtBytes(bytesTotal),
		fmtBytes(avgPage)))

	// Timing
	b.WriteString(fmt.Sprintf("  %s %s  \u2502  ETA %s  \u2502  Avg %dms/req  \u2502  In-flight %s\n",
		tuiLabelStyle.Render("Elapsed"),
		tuiValueStyle.Render(formatDuration(elapsed)),
		eta,
		int(m.stats.AvgFetchMs()),
		fmtInt64(inflight)))

	return b.String()
}

func (m tuiModel) viewResults() string {
	succ := m.stats.success.Load()
	fail := m.stats.failed.Load()
	tout := m.stats.timeout.Load()
	blk := m.stats.blocked.Load()
	skip := m.stats.skipped.Load()
	done := succ + fail + tout + blk + skip
	retryCount := m.stats.retries.Load()
	exhausted := m.stats.retryExhausted.Load()

	var b strings.Builder

	// Results breakdown
	result := fmt.Sprintf("  %s %s (%4.1f%%)  %s %s (%4.1f%%)  %s %s (%4.1f%%)",
		tuiSuccessStyle.Render("\u2713"), fmtInt64(succ), safePct(succ, done),
		tuiErrorStyle.Render("\u2717"), fmtInt64(fail), safePct(fail, done),
		tuiWarningStyle.Render("\u23f1"), fmtInt64(tout), safePct(tout, done))
	if blk > 0 {
		result += fmt.Sprintf("  \U0001f6ab %s blocked", fmtInt64(blk))
	}
	if skip > 0 {
		result += fmt.Sprintf("  \u23ed %s skipped", fmtInt64(skip))
	}
	if retryCount > 0 {
		result += fmt.Sprintf("  \u21bb %s retried", fmtInt64(retryCount))
	}
	if exhausted > 0 {
		result += fmt.Sprintf("  \u2718 %s gave up", fmtInt64(exhausted))
	}
	b.WriteString(result + "\n")

	// Retry queue
	if m.stats.retryQLen != nil {
		rqLen := m.stats.retryQLen()
		if rqLen > 0 || retryCount > 0 {
			b.WriteString(fmt.Sprintf("  %s %s pending  \u2502  max %d attempts\n",
				tuiLabelStyle.Render("RetryQ"),
				fmtInt(rqLen), maxRetryAttempts))
		}
	}

	// Frontier
	frontierQ := "---"
	bloomN := "---"
	if m.stats.frontierLen != nil {
		frontierQ = fmtInt(m.stats.frontierLen())
	}
	if m.stats.bloomCount != nil {
		bloomN = fmtInt(int(m.stats.bloomCount()))
	}
	frontierLine := fmt.Sprintf("  %s %s queued  \u2502  %s seen  \u2502  %s links",
		tuiLabelStyle.Render("Frontier"),
		frontierQ, bloomN, fmtInt64(m.stats.linksFound.Load()))
	if reseeds := m.stats.reseeds.Load(); reseeds > 0 {
		frontierLine += fmt.Sprintf("  \u2502  %s reseeds", fmtInt64(reseeds))
	}
	b.WriteString(frontierLine + "\n")

	return b.String()
}

func (m tuiModel) viewWorkers() string {
	var b strings.Builder

	workerLine := m.stats.rodPhaseLine()
	if restarts := m.stats.rodRestarts.Load(); restarts > 0 {
		workerLine += fmt.Sprintf("  \u2502  \u21bb %s restarts", fmtInt64(restarts))
	}
	b.WriteString(fmt.Sprintf("  %s %s\n", tuiLabelStyle.Render("Workers"), workerLine))

	if details := m.stats.rodWorkerDetails(); details != "" {
		b.WriteString(details)
	}
	return b.String()
}

func (m tuiModel) viewHTTPDepth() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("  %s %s\n", tuiLabelStyle.Render("HTTP"), m.stats.statusLine()))
	b.WriteString(fmt.Sprintf("  %s %s\n", tuiLabelStyle.Render("Depth"), m.stats.depthLine()))
	return b.String()
}
