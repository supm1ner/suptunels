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

	cfg, err := config.Load("")
	if err != nil {
		log.Fatal(err)
	}
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
	p := tea.NewProgram(tui.Model{
		Collector: collector,
		IsServer:  false,
	}, tea.WithAltScreen())

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
