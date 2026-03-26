package metrics

import (
	"sync"
	"time"
)

type TunnelStats struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	ExternalPort int       `json:"external_port"`
	InternalPort int       `json:"internal_port"`
	Type         string    `json:"type"`
	RX           uint64    `json:"rx"`
	TX           uint64    `json:"tx"`
	Conns        int       `json:"connections"`
	LastSeen     time.Time `json:"last_seen"`
	Uptime       time.Time `json:"uptime"`
	Status       string    `json:"status"` // "online", "offline"
}

type Collector struct {
	mu     sync.RWMutex
	stats  map[string]*TunnelStats
	startTime time.Time
}

func NewCollector() *Collector {
	return &Collector{
		stats:     make(map[string]*TunnelStats),
		startTime: time.Now(),
	}
}

func (c *Collector) UpdateRX(id string, n uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if s, ok := c.stats[id]; ok {
		s.RX += n
		s.LastSeen = time.Now()
	}
}

func (c *Collector) UpdateTX(id string, n uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if s, ok := c.stats[id]; ok {
		s.TX += n
		s.LastSeen = time.Now()
	}
}

func (c *Collector) SetStatus(id, name, status string, ext, int int, t string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.stats[id]; !ok {
		c.stats[id] = &TunnelStats{
			ID:           id,
			Name:         name,
			ExternalPort: ext,
			InternalPort: int,
			Type:         t,
			Uptime:       time.Now(),
		}
	}
	c.stats[id].Status = status
	if status == "online" {
		c.stats[id].Uptime = time.Now()
	}
}

func (c *Collector) IncConn(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if s, ok := c.stats[id]; ok {
		s.Conns++
	}
}

func (c *Collector) DecConn(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if s, ok := c.stats[id]; ok {
		s.Conns--
	}
}

func (c *Collector) GetStats() []TunnelStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	res := make([]TunnelStats, 0, len(c.stats))
	for _, s := range c.stats {
		res = append(res, *s)
	}
	return res
}

func (c *Collector) GetGlobalUptime() time.Duration {
	return time.Since(c.startTime)
}
