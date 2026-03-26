package tunnel

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/supminer/suptunnels/internal/config"
)

func (m *TunnelManager) HandleBoth(ctx context.Context, t config.TunnelConfig) {
	// TCP
	go m.HandleTCP(ctx, t)
	
	// UDP
	go m.HandleUDP(ctx, t)
}

func (m *TunnelManager) HandleAuto(ctx context.Context, t config.TunnelConfig) {
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
	m.collector.SetStatus(t.ID, t.Name, "online")

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

		go func(c net.Conn) {
			defer c.Close()
			buf := make([]byte, 16)
			n, err := c.Read(buf)
			if err != nil {
				return
			}

			proto := DetectProtocol(buf[:n])
			if proto == "udp" {
				log.Printf("[Info] Detected UDP on TCP port for %s", t.Name)
				// Not really supported over TCP listener, but we could wrap it.
				// For now, let's just treat as TCP.
			}
			
			// We need to pass the read bytes back
			m.proxyTCPWithInitial(ctx, c, t, buf[:n])
		}(conn)
	}
}

func (m *TunnelManager) proxyTCPWithInitial(ctx context.Context, conn net.Conn, t config.TunnelConfig, initial []byte) {
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

	// Handshake: 0 for TCP + tunnel ID
	idBytes := []byte(t.ID)
	idLen := len(idBytes)
	header := make([]byte, 5+idLen)
	header[0] = 0 // 0 for TCP
	header[1] = byte(idLen >> 24)
	header[2] = byte(idLen >> 16)
	header[3] = byte(idLen >> 8)
	header[4] = byte(idLen)
	copy(header[5:], idBytes)

	if _, err := stream.Write(header); err != nil {
		return
	}

	// Send the initial data we read for protocol detection
	if _, err := stream.Write(initial); err != nil {
		return
	}

	m.collector.IncConn(t.ID)
	defer m.collector.DecConn(t.ID)

	done := make(chan struct{})
	go func() {
		n, _ := m.copyStats(stream, conn, t.ID, false)
		m.collector.UpdateTX(t.ID, uint64(n))
		close(done)
	}()

	go func() {
		n, _ := m.copyStats(conn, stream, t.ID, true)
		m.collector.UpdateRX(t.ID, uint64(n))
		close(done)
	}()

	select {
	case <-ctx.Done():
	case <-done:
	}
}

func (m *TunnelManager) copyStats(dst io.Writer, src io.Reader, id string, isRX bool) (int64, error) {
	buf := make([]byte, 32*1024)
	var written int64
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
				if isRX {
					m.collector.UpdateRX(id, uint64(nw))
				} else {
					m.collector.UpdateTX(id, uint64(nw))
				}
			}
			if ew != nil {
				return written, ew
			}
			if nr != nw {
				return written, io.ErrShortWrite
			}
		}
		if er != nil {
			if er != io.EOF {
				return written, er
			}
			break
		}
	}
	return written, nil
}
