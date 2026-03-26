package tunnel

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/supminer/suptunnels/internal/config"
)

func (m *TunnelManager) HandleTCP(ctx context.Context, t config.TunnelConfig) {
	addr := fmt.Sprintf(":%d", t.ExternalPort)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("[Error] Failed to listen TCP on %s: %v", addr, err)
		return
	}
	defer l.Close()

	m.mu.Lock()
	m.active[t.ID] = &ActiveTunnel{Config: t, Listener: l}
	m.mu.Unlock()
	m.collector.SetStatus(t.ID, t.Name, "online", t.ExternalPort, t.InternalPort, t.Type)
	defer m.collector.SetStatus(t.ID, t.Name, "offline", t.ExternalPort, t.InternalPort, t.Type)

	for {
		conn, err := l.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				log.Printf("[Error] Accept TCP on %s: %v", addr, err)
				continue
			}
		}

		go m.proxyTCP(ctx, conn, t)
	}
}

func (m *TunnelManager) proxyTCP(ctx context.Context, conn net.Conn, t config.TunnelConfig) {
	defer conn.Close()
	m.mu.RLock()
	session := m.session
	m.mu.RUnlock()

	if session == nil {
		return
	}

	stream, err := session.OpenStream()
	if err != nil {
		log.Printf("[Error] OpenStream for %s: %v", t.Name, err)
		return
	}
	defer stream.Close()

	// Handshake to tell the client which tunnel this stream is for
	// Protocol: [4 bytes tunnel ID length][tunnel ID]
	idBytes := []byte(t.ID)
	idLen := len(idBytes)
	header := make([]byte, 4+idLen)
	header[0] = byte(idLen >> 24)
	header[1] = byte(idLen >> 16)
	header[2] = byte(idLen >> 8)
	header[3] = byte(idLen)
	copy(header[4:], idBytes)

	if _, err := stream.Write(header); err != nil {
		return
	}

	m.collector.IncConn(t.ID)
	defer m.collector.DecConn(t.ID)

	done := make(chan struct{})
	go func() {
		n, _ := io.Copy(stream, conn)
		m.collector.UpdateTX(t.ID, uint64(n))
		close(done)
	}()

	go func() {
		n, _ := io.Copy(conn, stream)
		m.collector.UpdateRX(t.ID, uint64(n))
		close(done)
	}()

	select {
	case <-ctx.Done():
	case <-done:
	}
}
