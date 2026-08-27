// Package event implements the Event Simulation Engine: a typed Event model
// plus a high-throughput, backpressure-aware Bus built on goroutines and
// channels so a Universe can push millions of synthetic events (Login,
// Purchase, API Request, Network Packet, ...) through the system without
// blocking the UI or the rest of the simulation.
package event

import (
	"time"
)

// Priority controls scheduling/consumption order hints for the Bus.
type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// Event is one unit of activity flowing through the simulation.
type Event struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"` // login, purchase, error, api_request, db_query, ...
	Source        string                 `json:"source"`
	Destination   string                 `json:"destination,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
	Payload       map[string]interface{} `json:"payload,omitempty"`
	Priority      Priority               `json:"priority"`
	CorrelationID string                 `json:"correlationId,omitempty"`
	Metadata      map[string]string      `json:"metadata,omitempty"`
}
