package herd

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/memberlist"
)

// NodeMeta is the metadata each node broadcasts via SWIM gossip.
type NodeMeta struct {
	DataAddr string `json:"a"` // TCP address for HD binary protocol
	Status   string `json:"s"` // "ready", "draining", "joining"
	Weight   int    `json:"w"` // Relative capacity weight (default 100)
}

// GossipConfig configures the memberlist-based gossip membership.
type GossipConfig struct {
	// NodeName is a unique name for this node (default: hostname).
	NodeName string
	// BindAddr is the gossip bind address (default: 0.0.0.0).
	BindAddr string
	// BindPort is the gossip bind port (default: 7241).
	BindPort int
	// DataAddr is the TCP address for HD binary protocol.
	DataAddr string
	// Seeds is the list of seed nodes to join.
	Seeds []string
	// OnJoin is called when a new node joins the cluster.
	OnJoin func(name string, meta NodeMeta)
	// OnLeave is called when a node leaves the cluster.
	OnLeave func(name string)
}

// Membership manages cluster membership using HashiCorp memberlist (SWIM protocol).
type Membership struct {
	list    *memberlist.Memberlist
	config  GossipConfig
	meta    NodeMeta
	metaMu  sync.RWMutex

	mu    sync.RWMutex
	nodes map[string]NodeMeta // name → meta
}

// NewMembership creates and starts a memberlist-based gossip membership.
func NewMembership(cfg GossipConfig) (*Membership, error) {
	m := &Membership{
		config: cfg,
		meta: NodeMeta{
			DataAddr: cfg.DataAddr,
			Status:   "ready",
			Weight:   100,
		},
		nodes: make(map[string]NodeMeta),
	}

	mlConfig := memberlist.DefaultLANConfig()
	mlConfig.Name = cfg.NodeName
	if mlConfig.Name == "" {
		mlConfig.Name = cfg.DataAddr
	}

	if cfg.BindAddr != "" {
		mlConfig.BindAddr = cfg.BindAddr
	}
	if cfg.BindPort > 0 {
		mlConfig.BindPort = cfg.BindPort
		mlConfig.AdvertisePort = cfg.BindPort
	} else {
		mlConfig.BindPort = 7241
		mlConfig.AdvertisePort = 7241
	}

	mlConfig.Delegate = m
	mlConfig.Events = m

	// Quiet memberlist logging.
	mlConfig.LogOutput = log.Writer()

	list, err := memberlist.Create(mlConfig)
	if err != nil {
		return nil, fmt.Errorf("herd: memberlist create: %w", err)
	}
	m.list = list

	// Join seed nodes if specified.
	if len(cfg.Seeds) > 0 {
		var seeds []string
		for _, s := range cfg.Seeds {
			s = strings.TrimSpace(s)
			if s != "" {
				seeds = append(seeds, s)
			}
		}
		if len(seeds) > 0 {
			n, err := list.Join(seeds)
			if err != nil {
				log.Printf("herd: gossip join warning (joined %d of %d seeds): %v", n, len(seeds), err)
			}
		}
	}

	return m, nil
}

// Members returns the current list of live nodes with their metadata.
func (m *Membership) Members() map[string]NodeMeta {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]NodeMeta, len(m.nodes))
	for k, v := range m.nodes {
		result[k] = v
	}
	return result
}

// LiveDataAddrs returns the TCP data addresses of all live "ready" nodes.
func (m *Membership) LiveDataAddrs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	addrs := make([]string, 0, len(m.nodes))
	for _, meta := range m.nodes {
		if meta.Status == "ready" && meta.DataAddr != "" {
			addrs = append(addrs, meta.DataAddr)
		}
	}
	return addrs
}

// SetStatus updates this node's status and triggers a gossip update.
func (m *Membership) SetStatus(status string) {
	m.metaMu.Lock()
	m.meta.Status = status
	m.metaMu.Unlock()
	m.list.UpdateNode(5 * time.Second)
}

// Leave gracefully leaves the cluster.
func (m *Membership) Leave(timeout time.Duration) error {
	return m.list.Leave(timeout)
}

// Shutdown stops the memberlist.
func (m *Membership) Shutdown() error {
	return m.list.Shutdown()
}

// NumMembers returns the number of known members.
func (m *Membership) NumMembers() int {
	return m.list.NumMembers()
}

// --- memberlist.Delegate interface ---

// NodeMeta returns this node's metadata for gossip.
func (m *Membership) NodeMeta(limit int) []byte {
	m.metaMu.RLock()
	defer m.metaMu.RUnlock()
	data, _ := json.Marshal(m.meta)
	if len(data) > limit {
		return nil
	}
	return data
}

// NotifyMsg handles incoming gossip messages (unused for now).
func (m *Membership) NotifyMsg([]byte) {}

// GetBroadcasts returns pending broadcasts (unused for now).
func (m *Membership) GetBroadcasts(overhead, limit int) [][]byte { return nil }

// LocalState returns local state for full sync (unused for now).
func (m *Membership) LocalState(join bool) []byte { return nil }

// MergeRemoteState merges remote state from full sync (unused for now).
func (m *Membership) MergeRemoteState(buf []byte, join bool) {}

// --- memberlist.EventDelegate interface ---

// NotifyJoin is called when a node joins the cluster.
func (m *Membership) NotifyJoin(node *memberlist.Node) {
	var meta NodeMeta
	if len(node.Meta) > 0 {
		json.Unmarshal(node.Meta, &meta)
	}
	// If no data addr in meta, construct from node address.
	if meta.DataAddr == "" {
		meta.DataAddr = net.JoinHostPort(node.Addr.String(), strconv.Itoa(int(node.Port)+1000))
	}
	if meta.Status == "" {
		meta.Status = "ready"
	}

	m.mu.Lock()
	m.nodes[node.Name] = meta
	m.mu.Unlock()

	if m.config.OnJoin != nil {
		m.config.OnJoin(node.Name, meta)
	}
}

// NotifyLeave is called when a node leaves the cluster.
func (m *Membership) NotifyLeave(node *memberlist.Node) {
	m.mu.Lock()
	delete(m.nodes, node.Name)
	m.mu.Unlock()

	if m.config.OnLeave != nil {
		m.config.OnLeave(node.Name)
	}
}

// NotifyUpdate is called when a node's metadata changes.
func (m *Membership) NotifyUpdate(node *memberlist.Node) {
	var meta NodeMeta
	if len(node.Meta) > 0 {
		json.Unmarshal(node.Meta, &meta)
	}
	if meta.DataAddr == "" {
		meta.DataAddr = net.JoinHostPort(node.Addr.String(), strconv.Itoa(int(node.Port)+1000))
	}

	m.mu.Lock()
	m.nodes[node.Name] = meta
	m.mu.Unlock()
}
