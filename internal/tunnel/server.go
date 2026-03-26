package tunnel

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/hashicorp/yamux"
	"github.com/supminer/suptunnels/internal/config"
	"github.com/supminer/suptunnels/internal/metrics"
)

type Server struct {
	cfg       *config.Config
	collector *metrics.Collector
	manager   *TunnelManager
}

func NewServer(cfg *config.Config, collector *metrics.Collector) *Server {
	return &Server{
		cfg:       cfg,
		collector: collector,
		manager:   NewManager(cfg, collector),
	}
}

func (s *Server) Start(ctx context.Context) error {
	l, err := net.Listen("tcp", s.cfg.Server.ControlAddr)
	if err != nil {
		return err
	}
	defer l.Close()

	log.Printf("[Info] Tunnel server listening on %s", s.cfg.Server.ControlAddr)

	for {
		conn, err := l.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				log.Printf("[Error] Accept connection: %v", err)
				continue
			}
		}

		go s.handleClient(ctx, conn)
	}
}

func (s *Server) handleClient(ctx context.Context, conn net.Conn) {
	// Authentication
	buf := make([]byte, len(s.cfg.Server.Secret))
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	if _, err := conn.Read(buf); err != nil || string(buf) != s.cfg.Server.Secret {
		log.Printf("[Error] Auth failed for %s", conn.RemoteAddr())
		conn.Close()
		return
	}
	conn.SetReadDeadline(time.Time{})
	log.Printf("[Info] Client authenticated: %s", conn.RemoteAddr())

	session, err := yamux.Server(conn, nil)
	if err != nil {
		log.Printf("[Error] Yamux server session: %v", err)
		return
	}
	defer session.Close()

	s.manager.SetSession(session)
	
	// Start tunnels
	log.Printf("[Info] Loading %d tunnels from config", len(s.cfg.Tunnels))
	for _, t := range s.cfg.Tunnels {
		if !t.Enabled {
			log.Printf("[Info] Tunnel %s is disabled, skipping", t.Name)
			continue
		}
		log.Printf("[Info] Starting tunnel %s (%s) on port %d", t.Name, t.Type, t.ExternalPort)
		switch t.Type {
		case "tcp":
			go s.manager.HandleTCP(ctx, t)
		case "udp":
			go s.manager.HandleUDP(ctx, t)
		case "both":
			go s.manager.HandleBoth(ctx, t)
		default:
			log.Printf("[Error] Unknown tunnel type: %s for %s", t.Type, t.Name)
		}
	}

	<-session.CloseChan()
	log.Printf("[Info] Client disconnected: %s", conn.RemoteAddr())
}
