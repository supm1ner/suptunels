package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/supminer/suptunnels/internal/metrics"
)

type Model struct {
	Collector *metrics.Collector
	Uptime    time.Duration
	Logs      []string
	Stats     []metrics.TunnelStats
	IsServer  bool
	Connected bool
}

type TickMsg time.Time

func (m Model) Init() tea.Cmd {
	return tick()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	case TickMsg:
		m.Stats = m.Collector.GetStats()
		m.Uptime = m.Collector.GetGlobalUptime()
		return m, tick()
	}
	return m, nil
}

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}
