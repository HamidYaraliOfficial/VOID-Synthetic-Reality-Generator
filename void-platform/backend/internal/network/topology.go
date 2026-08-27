// Package network implements the Network Topology Simulator: a purely
// synthetic graph of Router/Server/Client/Gateway/LoadBalancer/Firewall/
// Service/Segment nodes whose links carry configurable latency, packet loss,
// bandwidth and jitter. Nothing in this package ever touches a real network
// interface — it only shapes numbers used to simulate how a system *would*
// behave under those conditions, and to feed the Service Dependency Graph.
package network

import (
	"math/rand"
	"sync"
	"time"
)

type NodeKind string

const (
	NodeRouter      NodeKind = "router"
	NodeServer      NodeKind = "server"
	NodeClient      NodeKind = "client"
	NodeGateway     NodeKind = "gateway"
	NodeLoadBalancer NodeKind = "load_balancer"
	NodeFirewall    NodeKind = "firewall"
	NodeService     NodeKind = "service"
	NodeSegment     NodeKind = "segment"
	NodeDatabase    NodeKind = "database"
	NodeCache       NodeKind = "cache"
)

// Node is one synthetic network/service participant.
type Node struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Kind      NodeKind `json:"kind"`
	Available bool     `json:"available"`
}

// Link is a directed synthetic connection between two Nodes with tunable
// characteristics used by the Chaos engine and the Load Generator to derive
// simulated response times.
type Link struct {
	From        string        `json:"from"`
	To          string        `json:"to"`
	LatencyMs   float64       `json:"latencyMs"`
	JitterMs    float64       `json:"jitterMs"`
	PacketLoss  float64       `json:"packetLoss"`  // 0..1
	BandwidthKbps float64     `json:"bandwidthKbps"`
	Up          bool          `json:"up"`
}

// Topology is a concurrency-safe synthetic network/service graph.
type Topology struct {
	mu    sync.RWMutex
	Nodes map[string]*Node
	Links map[string]*Link // keyed "from->to"
}

func NewTopology() *Topology {
	return &Topology{Nodes: map[string]*Node{}, Links: map[string]*Link{}}
}

func (t *Topology) AddNode(n *Node) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if n.ID == "" {
		return
	}
	n.Available = true
	t.Nodes[n.ID] = n
}

func linkKey(from, to string) string { return from + "->" + to }

func (t *Topology) AddLink(l *Link) {
	t.mu.Lock()
	defer t.mu.Unlock()
	l.Up = true
	t.Links[linkKey(l.From, l.To)] = l
}

// SetLatency implements the chaos.Target interface: mutate a link's latency
// at runtime (e.g. from a "high latency" chaos action).
func (t *Topology) SetLatency(from, to string, ms float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if l, ok := t.Links[linkKey(from, to)]; ok {
		l.LatencyMs = ms
	}
}

// SetPacketLoss mutates a link's packet loss ratio (0..1) at runtime.
func (t *Topology) SetPacketLoss(from, to string, ratio float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if l, ok := t.Links[linkKey(from, to)]; ok {
		l.PacketLoss = ratio
	}
}

// SetNodeAvailability marks a synthetic node as up/down, used to simulate
// "Service Unavailable" style chaos and to drive cascade-failure math in the
// Service Dependency Graph.
func (t *Topology) SetNodeAvailability(id string, available bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if n, ok := t.Nodes[id]; ok {
		n.Available = available
	}
}

// SimulateHop computes a synthetic (latency, dropped) result for traversing
// one link, honouring configured jitter and packet loss.
func (t *Topology) SimulateHop(r *rand.Rand, from, to string) (latency time.Duration, dropped bool) {
	t.mu.RLock()
	l, ok := t.Links[linkKey(from, to)]
	t.mu.RUnlock()
	if !ok || !l.Up {
		return 0, true
	}
	if l.PacketLoss > 0 && r.Float64() < l.PacketLoss {
		return 0, true
	}
	jitter := 0.0
	if l.JitterMs > 0 {
		jitter = (r.Float64()*2 - 1) * l.JitterMs
	}
	ms := l.LatencyMs + jitter
	if ms < 0 {
		ms = 0
	}
	return time.Duration(ms * float64(time.Millisecond)), false
}

// Snapshot returns a copy of nodes/links for export/inspection/UI rendering.
func (t *Topology) Snapshot() ([]*Node, []*Link) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	nodes := make([]*Node, 0, len(t.Nodes))
	for _, n := range t.Nodes {
		cp := *n
		nodes = append(nodes, &cp)
	}
	links := make([]*Link, 0, len(t.Links))
	for _, l := range t.Links {
		cp := *l
		links = append(links, &cp)
	}
	return nodes, links
}

// CascadeImpact walks outgoing links from a failed node and returns the IDs
// of directly and transitively dependent nodes whose effective availability
// is impacted — implementing the "Service Dependency Graph" cascade-failure
// calculation from the product spec (simple reachability, not full graph
// theory, which is sufficient for a synthetic dependency fan-out).
func (t *Topology) CascadeImpact(failedNodeID string) []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	visited := map[string]bool{failedNodeID: true}
	queue := []string{failedNodeID}
	var impacted []string
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for key, l := range t.Links {
			_ = key
			if l.From == cur && !visited[l.To] {
				visited[l.To] = true
				impacted = append(impacted, l.To)
				queue = append(queue, l.To)
			}
		}
	}
	return impacted
}
