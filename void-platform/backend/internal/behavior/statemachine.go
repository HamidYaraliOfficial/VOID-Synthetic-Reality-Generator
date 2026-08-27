// Package behavior implements the Behavior Engine: State Machine / Behavior
// Tree style definitions that drive how an Entity acts over time (a User
// logging in, browsing, purchasing, retrying on error, ...). Definitions are
// authored visually in the frontend Behavior Editor as a graph of
// Event/Condition/Probability/Action/Delay/State-Change nodes and shipped to
// the engine as the JSON structures below.
package behavior

import (
	"context"
	"math/rand"
	"time"
)

// NodeKind enumerates the node palette exposed by the Visual Behavior Editor.
type NodeKind string

const (
	NodeEvent        NodeKind = "event"
	NodeCondition    NodeKind = "condition"
	NodeProbability  NodeKind = "probability"
	NodeAction       NodeKind = "action"
	NodeDelay        NodeKind = "delay"
	NodeStateChange  NodeKind = "state_change"
	NodeAPICall      NodeKind = "api_call"
	NodeDBOperation  NodeKind = "db_operation"
	NodeLoop         NodeKind = "loop"
)

// Node is one vertex of a Behavior Graph.
type Node struct {
	ID         string                 `json:"id"`
	Kind       NodeKind               `json:"kind"`
	Params     map[string]interface{} `json:"params,omitempty"`
	Next       []string               `json:"next,omitempty"`       // node IDs to follow on success/default
	OnFailure  []string               `json:"onFailure,omitempty"`  // node IDs to follow on failure/condition-false
	LoopTarget string                 `json:"loopTarget,omitempty"` // for NodeLoop: node ID to jump back to
	LoopCount  int                    `json:"loopCount,omitempty"`
}

// Graph is a full Behavior definition: a set of Nodes plus the entry point.
type Graph struct {
	Name    string          `json:"name"`
	Entry   string          `json:"entry"`
	Nodes   map[string]Node `json:"nodes"`
	States  []string        `json:"states,omitempty"`
}

// RunContext carries everything a single execution of a Graph needs: the
// entity id/type driving it, a place to stash transient variables, and
// callbacks the simulation engine wires up (emit event, call API, do a DB
// op, change entity state) without behavior needing to import simulation.
type RunContext struct {
	Context     context.Context
	EntityID    string
	EntityType  string
	RNG         *rand.Rand
	Vars        map[string]interface{}
	EmitEvent   func(nodeParams map[string]interface{})
	DoAPICall   func(nodeParams map[string]interface{}) error
	DoDBOp      func(nodeParams map[string]interface{}) error
	ChangeState func(newState string)
	Sleep       func(d time.Duration)
}

// Runner executes a Graph step by step for one entity. It is intentionally
// simple/interpreted (no compilation step) so behaviors authored visually in
// the frontend can be hot-reloaded onto already-running simulations.
type Runner struct {
	Graph *Graph
}

func NewRunner(g *Graph) *Runner {
	return &Runner{Graph: g}
}

// Run walks the graph starting at Entry until it reaches a node with no
// "next" edges, the loop budget is exhausted, or ctx is cancelled. maxSteps
// guards against accidental infinite loops in a user-authored graph.
func (rn *Runner) Run(rc *RunContext, maxSteps int) error {
	if rn.Graph == nil || rn.Graph.Entry == "" {
		return nil
	}
	current := rn.Graph.Entry
	steps := 0
	loopCounters := map[string]int{}

	for current != "" {
		select {
		case <-rc.Context.Done():
			return rc.Context.Err()
		default:
		}
		steps++
		if steps > maxSteps {
			return nil // step budget exhausted; treat as graceful completion
		}
		node, ok := rn.Graph.Nodes[current]
		if !ok {
			return nil
		}
		nextID, err := rn.execNode(rc, node, loopCounters)
		if err != nil && len(node.OnFailure) > 0 {
			current = node.OnFailure[0]
			continue
		}
		current = nextID
	}
	return nil
}

func (rn *Runner) execNode(rc *RunContext, node Node, loopCounters map[string]int) (string, error) {
	switch node.Kind {
	case NodeEvent:
		if rc.EmitEvent != nil {
			rc.EmitEvent(node.Params)
		}
		return firstOr(node.Next), nil

	case NodeCondition:
		if evalCondition(rc, node.Params) {
			return firstOr(node.Next), nil
		}
		return firstOr(node.OnFailure), nil

	case NodeProbability:
		p := 0.5
		if v, ok := node.Params["p"].(float64); ok {
			p = v
		}
		if rc.RNG.Float64() < p {
			return firstOr(node.Next), nil
		}
		return firstOr(node.OnFailure), nil

	case NodeAction:
		// Generic no-op action hook; kept for extension via plugins.
		return firstOr(node.Next), nil

	case NodeDelay:
		d := time.Duration(0)
		if v, ok := node.Params["ms"].(float64); ok {
			d = time.Duration(v) * time.Millisecond
		}
		if rc.Sleep != nil && d > 0 {
			rc.Sleep(d)
		}
		return firstOr(node.Next), nil

	case NodeStateChange:
		if s, ok := node.Params["state"].(string); ok && rc.ChangeState != nil {
			rc.ChangeState(s)
		}
		return firstOr(node.Next), nil

	case NodeAPICall:
		if rc.DoAPICall != nil {
			if err := rc.DoAPICall(node.Params); err != nil {
				return firstOr(node.OnFailure), err
			}
		}
		return firstOr(node.Next), nil

	case NodeDBOperation:
		if rc.DoDBOp != nil {
			if err := rc.DoDBOp(node.Params); err != nil {
				return firstOr(node.OnFailure), err
			}
		}
		return firstOr(node.Next), nil

	case NodeLoop:
		key := node.ID
		limit := node.LoopCount
		if limit <= 0 {
			limit = 1
		}
		loopCounters[key]++
		if loopCounters[key] <= limit {
			return node.LoopTarget, nil
		}
		return firstOr(node.Next), nil

	default:
		return firstOr(node.Next), nil
	}
}

func firstOr(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	return ids[0]
}

// evalCondition supports a small, safe set of comparisons against rc.Vars so
// authors don't need a full expression language for simple branching.
func evalCondition(rc *RunContext, params map[string]interface{}) bool {
	field, _ := params["field"].(string)
	op, _ := params["op"].(string)
	target := params["value"]
	if field == "" {
		return true
	}
	current, ok := rc.Vars[field]
	if !ok {
		return false
	}
	cf, cok := toFloat(current)
	tf, tok := toFloat(target)
	if cok && tok {
		switch op {
		case "<":
			return cf < tf
		case "<=":
			return cf <= tf
		case ">":
			return cf > tf
		case ">=":
			return cf >= tf
		case "==":
			return cf == tf
		case "!=":
			return cf != tf
		}
	}
	// fallback: string equality
	switch op {
	case "==":
		return current == target
	case "!=":
		return current != target
	}
	return false
}

func toFloat(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}
