package ui

import (
	"fmt"
	"sync"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"mkvtea/internal/checkpoint"
	"mkvtea/internal/config"
)

// ProcessModel represents the TUI state during file processing
type ProcessModel struct {
	// Config
	cfg   config.Config
	files []string

	// Progress tracking
	totalFiles   int
	processedIdx int
	successCount int
	skippedCount int
	errorCount   int

	// UI components
	spinner  spinner.Model
	viewport viewport.Model
	logs     []string
	mu       sync.Mutex

	// Window size
	width  int
	height int

	// DRY-RUN tracking
	extractedPaths []string // Paths where files would be extracted/merged
	outputDir      string   // Final output directory for merge mode

	// Concurrency
	sem chan struct{}
	wg  sync.WaitGroup

	// State
	finished      bool
	quitting      bool
	autoCloseTime time.Time

	// Checkpoint tracking
	checkpointMgr     *checkpoint.Manager
	checkpointCounter int
}

// NewProcessModel creates a new processor model
func NewProcessModel(cfg config.Config, files []string) *ProcessModel {
	s := spinner.New()
	// Classic rotating braille spinner
	s.Spinner = spinner.Spinner{
		Frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		FPS:    80,
	}
	s.Style = subtitleStyle

	vp := viewport.New(viewport.WithWidth(80), viewport.WithHeight(15))

	// Initialize checkpoint manager
	logs := []string{}
	checkpointMgr, err := checkpoint.NewManager(cfg)
	if err != nil {
		logs = append(logs, fmt.Sprintf("⚠️ CHECKPOINT: failed to initialize manager: %v", err))
	}

	return &ProcessModel{
		cfg:           cfg,
		files:         files,
		totalFiles:    len(files),
		spinner:       s,
		viewport:      vp,
		logs:          logs,
		sem:           make(chan struct{}, cfg.MaxProcs),
		processedIdx:  0,
		finished:      false,
		quitting:      false,
		autoCloseTime: time.Time{},
		width:         80, // Default, will be updated by WindowSizeMsg
		height:        24, // Default, will be updated by WindowSizeMsg
		checkpointMgr: checkpointMgr,
	}
}

func (m *ProcessModel) logCheckpointWarning(format string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCheckpointWarningLocked(format, args...)
}

func (m *ProcessModel) logCheckpointWarningLocked(format string, args ...any) {
	m.logs = append(m.logs, fmt.Sprintf("⚠️ CHECKPOINT: "+format, args...))
}

func (m *ProcessModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.startProcessing())
}

// ProcessingDoneMsg signals when all file processing is complete
type ProcessingDoneMsg struct{}

// AutoCloseMsg signals that the TUI should auto-close
type AutoCloseMsg struct{}

func (m *ProcessModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			m.mu.Lock()
			m.quitting = true
			m.mu.Unlock()
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		// Worker goroutines also touch the viewport under m.mu.
		m.mu.Lock()
		m.width = msg.Width
		m.height = msg.Height
		// Adjust viewport size based on window
		viewportHeight := msg.Height - 13 // Reserve space for header, stats, progress, footer
		if viewportHeight < 3 {
			viewportHeight = 3
		}
		m.viewport.SetWidth(msg.Width - 4) // Reserve space for padding/borders
		m.viewport.SetHeight(viewportHeight)
		m.mu.Unlock()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case ProcessingDoneMsg:
		m.mu.Lock()
		m.finished = true
		// Auto-close after the shared delay (see startAutoClose)
		m.autoCloseTime = time.Now().Add(autoCloseDelay)
		m.mu.Unlock()
		return m, m.startAutoClose()

	case AutoCloseMsg:
		m.mu.Lock()
		m.quitting = true
		m.mu.Unlock()
		return m, tea.Quit
	}

	return m, nil
}
