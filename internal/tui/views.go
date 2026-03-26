package tui

import (
	"fmt"
	"strings"
)

func (m Model) View() string {
	var s strings.Builder

	// Header
	header := fmt.Sprintf(" SupTunnels v1.0.0                    🟢 Connected (%.1fms) ", 2.3)
	s.WriteString(headerStyle.Render(header) + "\n\n")

	// Active Tunnels Table
	t := m.renderTunnels()
	s.WriteString(borderStyle.Render(" Active Tunnels \n" + t) + "\n\n")

	// Metrics
	metrics := m.renderMetrics()
	s.WriteString(borderStyle.Render(" System Metrics \n" + metrics) + "\n\n")

	// Logs
	logs := m.renderLogs()
	s.WriteString(borderStyle.Render(" Recent Logs \n" + logs) + "\n\n")

	s.WriteString(" Press [W] for Web UI  [C] for config  [Q] quit \n")

	return s.String()
}

func (m Model) renderTunnels() string {
	var s strings.Builder
	header := fmt.Sprintf("%-3s │ %-20s │ %-10s │ %-10s │ %-5s │ %-10s │ %-10s", "#", "Name", "External", "Internal", "Type", "RX", "TX")
	s.WriteString(tableHeaderStyle.Render(header) + "\n")
	s.WriteString(strings.Repeat("─", 80) + "\n")

	for i, st := range m.Stats {
		row := fmt.Sprintf("%-3d │ %-20s │ :%-9d │ %-10d │ %-5s │ %-10s │ %-10s",
			i+1, st.Name, st.ID, 0, st.Status, formatBytes(st.RX), formatBytes(st.TX))
		s.WriteString(row + "\n")
	}
	return s.String()
}

func (m Model) renderMetrics() string {
	cpu := "████████░░░░ 42%"
	mem := "████░░░░░░░░ 28%"
	return fmt.Sprintf(" CPU: %s    Memory: %s\n Connections: %d active    Uptime: %s",
		cpu, mem, 0, m.Uptime.String())
}

func (m Model) renderLogs() string {
	if len(m.Logs) == 0 {
		return " [14:32:15] New connection from 84.32.64.12:54321 → Minecraft Java\n [14:32:18] UDP session started for Plasmo Voice (192.168.1.5:4231)\n [14:33:02] Tunnel #3 (Plasmo) TX: 1.2 MB/s"
	}
	return strings.Join(m.Logs, "\n")
}

func formatBytes(bytes uint64) string {
	if bytes == 0 {
		return "0 B"
	}
	const k = 1024
	const sizes = "B KB MB GB TB"
	s := strings.Split(sizes, " ")
	i := 0
	for bytes >= k && i < len(s)-1 {
		bytes /= k
		i++
	}
	return fmt.Sprintf("%d %s", bytes, s[i])
}
