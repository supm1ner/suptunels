package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/supminer/suptunnels/internal/api"
	"github.com/supminer/suptunnels/internal/config"
	"github.com/supminer/suptunnels/internal/metrics"
	"github.com/supminer/suptunnels/internal/tui"
	"github.com/supminer/suptunnels/internal/tunnel"
	"github.com/supminer/suptunnels/internal/web"
)

func main() {
	port := flag.String("port", "8080", "Web UI port")
	controlPort := flag.String("tunnel-port", "8081", "Tunnel control port")
	secret := flag.String("secret", "supersecret", "API Secret")
	publicIP := flag.String("public-ip", "0.0.0.0", "Public IP")
	flag.Parse()

	cfg, err := config.Load("")
	if err != nil {
		log.Fatal(err)
	}
	cfg.Server.ListenAddr = ":" + *port
	cfg.Server.ControlAddr = ":" + *controlPort
	cfg.Server.Secret = *secret
	cfg.Server.PublicAddr = *publicIP

	collector := metrics.NewCollector()
	srv := tunnel.NewServer(cfg, collector)
	apiSrv := api.NewAPI(cfg, collector)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start tunnel server
	go func() {
		if err := srv.Start(ctx); err != nil {
			log.Printf("[Error] Tunnel server: %v", err)
		}
	}()

	// Start API and Web UI
	mux := http.NewServeMux()
	mux.Handle("/api/", apiSrv.Handlers())
	mux.Handle("/", web.Handler())
	
	webSrv := &http.Server{Addr: cfg.Server.ListenAddr, Handler: mux}
	go func() {
		log.Printf("[Info] Web UI started at http://localhost%s", cfg.Server.ListenAddr)
		if err := webSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[Error] Web UI: %v", err)
		}
	}()

	// Start TUI
	p := tea.NewProgram(tui.Model{
		Collector: collector,
		IsServer:  true,
	}, tea.WithAltScreen())

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		cancel()
		webSrv.Shutdown(context.Background())
		p.Quit()
	}()

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
