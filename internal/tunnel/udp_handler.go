package tunnel

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/supminer/suptunnels/internal/config"
)

type UDPSession struct {
	remoteAddr *net.UDPAddr
	lastSeen   time.Time
	stream     net.Conn
	done       chan struct{}
}

func (m *TunnelManager) HandleUDP(ctx context.Context, t config.TunnelConfig) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", t.ExternalPort))
	if err != nil {
		log.Printf("[Error] ResolveUDPAddr: %v", err)
		return
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Printf("[Error] ListenUDP on %d: %v", t.ExternalPort, err)
		return
	}
	defer conn.Close()

	m.mu.Lock()
	m.active[t.ID] = &ActiveTunnel{Config: t, UDPConn: conn}
	m.mu.Unlock()

	sessions := make(map[string]*UDPSession)
	var mu sync.Mutex

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				mu.Lock()
				for k, s := range sessions {
					if time.Since(s.lastSeen) > 5*time.Minute {
						s.stream.Close()
						delete(sessions, k)
					}
				}
				mu.Unlock()
			}
		}
	}()

	buf := make([]byte, 65535)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				log.Printf("[Error] ReadFromUDP: %v", err)
				continue
			}
		}

		key := remoteAddr.String()
		mu.Lock()
		s, ok := sessions[key]
		if !ok {
			stream, err := m.openUDPStream(t.ID)
			if err != nil {
				log.Printf("[Error] openUDPStream: %v", err)
				mu.Unlock()
				continue
			}

			s = &UDPSession{
				remoteAddr: remoteAddr,
				lastSeen:   time.Now(),
				stream:     stream,
				done:       make(chan struct{}),
			}
			sessions[key] = s
			m.collector.IncConn(t.ID)

			go func(session *UDPSession, key string) {
				buf := make([]byte, 65535)
				for {
					n, err := session.stream.Read(buf)
					if err != nil {
						mu.Lock()
						delete(sessions, key)
						mu.Unlock()
						m.collector.DecConn(t.ID)
						return
					}
					conn.WriteToUDP(buf[:n], session.remoteAddr)
					m.collector.UpdateRX(t.ID, uint64(n))
				}
			}(s, key)
		}
		s.lastSeen = time.Now()
		mu.Unlock()

		_, err = s.stream.Write(buf[:n])
		if err != nil {
			log.Printf("[Error] Write to UDP stream: %v", err)
		}
		m.collector.UpdateTX(t.ID, uint64(n))
	}
}

func (m *TunnelManager) openUDPStream(tunnelID string) (net.Conn, error) {
	m.mu.RLock()
	session := m.session
	m.mu.RUnlock()

	if session == nil {
		return nil, fmt.Errorf("no active session")
	}

	stream, err := session.OpenStream()
	if err != nil {
		return nil, err
	}

	// Handshake: UDP marker + tunnel ID
	idBytes := []byte(tunnelID)
	idLen := len(idBytes)
	header := make([]byte, 5+idLen)
	header[0] = 1 // 0 for TCP, 1 for UDP
	header[1] = byte(idLen >> 24)
	header[2] = byte(idLen >> 16)
	header[3] = byte(idLen >> 8)
	header[4] = byte(idLen)
	copy(header[5:], idBytes)

	if _, err := stream.Write(header); err != nil {
		stream.Close()
		return nil, err
	}

	return stream, nil
}
