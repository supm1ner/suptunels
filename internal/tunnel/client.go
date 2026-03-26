package tunnel

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/hashicorp/yamux"
	"github.com/supminer/suptunnels/internal/config"
	"github.com/supminer/suptunnels/internal/metrics"
)

type Client struct {
	cfg       *config.Config
	collector *metrics.Collector
	session   *yamux.Session
	mu        sync.RWMutex
}

func NewClient(cfg *config.Config, collector *metrics.Collector) *Client {
	return &Client{
		cfg:       cfg,
		collector: collector,
	}
}

func (c *Client) Start(ctx context.Context, serverAddr string, secret string) error {
	for {
		err := c.connect(ctx, serverAddr, secret)
		if err != nil {
			log.Printf("[Error] Connect to server %s: %v", serverAddr, err)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(5 * time.Second):
				continue
			}
		}
	}
}

func (c *Client) connect(ctx context.Context, serverAddr string, secret string) error {
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Authentication
	if _, err := conn.Write([]byte(secret)); err != nil {
		return err
	}

	session, err := yamux.Client(conn, nil)
	if err != nil {
		return err
	}
	defer session.Close()

	log.Printf("[Info] Connected to server: %s", serverAddr)

	for {
		stream, err := session.AcceptStream()
		if err != nil {
			return err
		}

		go c.handleStream(ctx, stream)
	}
}

func (c *Client) handleStream(ctx context.Context, stream *yamux.Stream) {
	defer stream.Close()

	// Read handshake
	// [1 byte proto: 0=TCP, 1=UDP][4 bytes tunnel ID length][tunnel ID]
	buf := make([]byte, 5)
	if _, err := io.ReadFull(stream, buf); err != nil {
		return
	}

	isUDP := buf[0] == 1
	idLen := int(uint32(buf[1])<<24 | uint32(buf[2])<<16 | uint32(buf[3])<<8 | uint32(buf[4]))
	idBuf := make([]byte, idLen)
	if _, err := io.ReadFull(stream, idBuf); err != nil {
		return
	}
	tunnelID := string(idBuf)

	// Find tunnel config by ID or use a default mapping
	var t config.TunnelConfig
	found := false
	for _, tc := range c.cfg.Tunnels {
		if tc.ID == tunnelID {
			t = tc
			found = true
			break
		}
	}

	// Fallback: if ID not found, check if it's a numeric port ID (common for manual configs)
	if !found {
		for _, tc := range c.cfg.Tunnels {
			if fmt.Sprintf("%d", tc.ExternalPort) == tunnelID {
				t = tc
				found = true
				break
			}
		}
	}

	if !found {
		log.Printf("[Error] Tunnel not found for ID: %s. Using default internal port from ID if possible.", tunnelID)
		// If ID is a number, try to use it as port directly
		var p int
		if _, err := fmt.Sscanf(tunnelID, "%d", &p); err == nil {
			t = config.TunnelConfig{Name: "Auto", InternalPort: p}
			found = true
		} else {
			return
		}
	}

	var localConn net.Conn
	var err error
	if isUDP {
		addr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", t.InternalPort))
		localConn, err = net.DialUDP("udp", nil, addr)
	} else {
		localConn, err = net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", t.InternalPort))
	}

	if err != nil {
		log.Printf("[Error] Connect to local %s:%d: %v", t.Name, t.InternalPort, err)
		return
	}
	defer localConn.Close()

	done := make(chan struct{})
	go func() {
		io.Copy(stream, localConn)
		close(done)
	}()
	go func() {
		io.Copy(localConn, stream)
		close(done)
	}()

	select {
	case <-ctx.Done():
	case <-done:
	}
}
