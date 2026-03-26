package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/supminer/suptunnels/internal/config"
	"github.com/supminer/suptunnels/internal/metrics"
	"github.com/supminer/suptunnels/internal/tui"
	"github.com/supminer/suptunnels/internal/tunnel"
)

func main() {
	serverAddr := flag.String("server", "localhost:8080", "Server address")
	secret := flag.String("secret", "supersecret", "API Secret")
	flag.Parse()

	// Redirect logs to TUI
	logChan := make(chan string, 100)
	log.SetOutput(&logWriter{ch: logChan})

	cfg, cfgPath, err := config.Load("")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("[Info] Config loaded from %s", cfgPath)
	cfg.Server.Secret = *secret

	collector := metrics.NewCollector()
	cli := tunnel.NewClient(cfg, collector)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start tunnel client
	go func() {
		if err := cli.Start(ctx, *serverAddr, *secret); err != nil {
			log.Printf("[Error] Tunnel client: %v", err)
		}
	}()

	// Start TUI
	model := tui.Model{
		Collector: collector,
		IsServer:  false,
		LogChan:   logChan,
	}
	p := tea.NewProgram(model, tea.WithAltScreen())

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		cancel()
		p.Quit()
	}()

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

type logWriter struct {
	ch chan string
}

func (w *logWriter) Write(p []byte) (n int, err error) {
	w.ch <- string(p)
	return len(p), nil
}
