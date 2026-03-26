package tunnel

import (
	"context"
	"io"
	"net"
	"sync"

	"github.com/hashicorp/yamux"
	"github.com/supminer/suptunnels/internal/config"
	"github.com/supminer/suptunnels/internal/metrics"
)

type TunnelManager struct {
	ctx        context.Context
	cancel     context.CancelFunc
	cfg        *config.Config
	collector  *metrics.Collector
	active     map[string]*ActiveTunnel
	mu         sync.RWMutex
	session    *yamux.Session
}

type ActiveTunnel struct {
	Config  config.TunnelConfig
	Cancel  context.CancelFunc
	Listener net.Listener
	UDPConn  *net.UDPConn
}

func NewManager(cfg *config.Config, collector *metrics.Collector) *TunnelManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &TunnelManager{
		ctx:       ctx,
		cancel:    cancel,
		cfg:       cfg,
		collector: collector,
		active:    make(map[string]*ActiveTunnel),
	}
}

func (m *TunnelManager) Stop() {
	m.cancel()
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.active {
		if t.Cancel != nil {
			t.Cancel()
		}
		if t.Listener != nil {
			t.Listener.Close()
		}
		if t.UDPConn != nil {
			t.UDPConn.Close()
		}
	}
}

func (m *TunnelManager) SetSession(session *yamux.Session) {
	m.mu.Lock()
	m.session = session
	m.mu.Unlock()
}

func Proxy(ctx context.Context, src, dst io.ReadWriteCloser, tunnelID string, collector *metrics.Collector, isRX bool) {
	defer src.Close()
	defer dst.Close()

	done := make(chan struct{})
	
	go func() {
		n, _ := io.Copy(src, dst)
		if isRX {
			collector.UpdateRX(tunnelID, uint64(n))
		} else {
			collector.UpdateTX(tunnelID, uint64(n))
		}
		close(done)
	}()

	go func() {
		n, _ := io.Copy(dst, src)
		if isRX {
			collector.UpdateTX(tunnelID, uint64(n))
		} else {
			collector.UpdateRX(tunnelID, uint64(n))
		}
		close(done)
	}()

	select {
	case <-ctx.Done():
	case <-done:
	}
}

func DetectProtocol(firstBytes []byte) string {
	if len(firstBytes) == 0 {
		return "tcp"
	}
	// Minecraft Java Handshake: 0x00
	if firstBytes[0] == 0x00 {
		return "tcp"
	}
	// RakNet (Bedrock/Plasmo): 0x01-0x06
	if firstBytes[0] >= 0x01 && firstBytes[0] <= 0x06 {
		return "udp"
	}
	// HTTP
	if len(firstBytes) > 3 && (firstBytes[0] == 'G' || firstBytes[0] == 'P' || firstBytes[0] == 'H') {
		return "tcp"
	}
	return "tcp"
}
