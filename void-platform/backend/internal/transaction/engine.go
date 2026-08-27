package transaction

import (
	"fmt"
	"sync"
	"time"
)

// State enumerates the lifecycle a synthetic transaction moves through.
type State string

const (
	StatePending  State = "pending"
	StateApproved State = "approved"
	StateRejected State = "rejected"
	StateCancelled State = "cancelled"
	StateCompleted State = "completed"
)

// Transaction is a synthetic business operation (payment, order, booking, ...).
type Transaction struct {
	ID        string                 `json:"id"`
	Kind      string                 `json:"kind"` // payment | order | reservation | inventory
	Fields    map[string]interface{} `json:"fields"`
	State     State                  `json:"state"`
	Reason    string                 `json:"reason,omitempty"`
	CreatedAt time.Time              `json:"createdAt"`
}

// Ledger is a concurrency-safe collection of synthetic account balances used
// by transaction rules like "if balance < amount then reject".
type Ledger struct {
	mu       sync.Mutex
	balances map[string]float64
	stock    map[string]int
}

func NewLedger() *Ledger {
	return &Ledger{balances: map[string]float64{}, stock: map[string]int{}}
}

func (l *Ledger) SetBalance(account string, amount float64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.balances[account] = amount
}

func (l *Ledger) Balance(account string) float64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.balances[account]
}

func (l *Ledger) SetStock(sku string, qty int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.stock[sku] = qty
}

func (l *Ledger) Stock(sku string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.stock[sku]
}

// Apply debits/credits an account by delta, clamping is not performed here;
// use the Rules Engine beforehand to reject transactions that would violate
// domain invariants (e.g. balance going negative).
func (l *Ledger) Apply(account string, delta float64) float64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.balances[account] += delta
	return l.balances[account]
}

// Processor ties the Rules Engine + Ledger together to process synthetic
// transactions the way a real payments/order system would validate them.
type Processor struct {
	Rules  *Engine
	Ledger *Ledger
}

func NewProcessor(rules *Engine, ledger *Ledger) *Processor {
	return &Processor{Rules: rules, Ledger: ledger}
}

// Process evaluates business rules for a transaction and, if not rejected,
// applies the balance/stock effects and marks it Completed.
func (p *Processor) Process(tx *Transaction) error {
	if p.Rules != nil {
		if ruleName, action := p.Rules.Evaluate(tx.Fields); action != "" {
			tx.Reason = fmt.Sprintf("rule %q matched", ruleName)
			switch action {
			case "reject":
				tx.State = StateRejected
				return nil
			case "cancel":
				tx.State = StateCancelled
				return nil
			case "flag":
				tx.State = StatePending
				return nil
			}
		}
	}

	if p.Ledger != nil {
		if account, ok := tx.Fields["account"].(string); ok {
			if amount, ok := toFloat(tx.Fields["amount"]); ok {
				sign := -1.0
				if tx.Kind == "refund" || tx.Kind == "deposit" {
					sign = 1.0
				}
				p.Ledger.Apply(account, sign*amount)
			}
		}
		if sku, ok := tx.Fields["sku"].(string); ok {
			if qty, ok := toFloat(tx.Fields["quantity"]); ok {
				p.Ledger.mu.Lock()
				p.Ledger.stock[sku] -= int(qty)
				p.Ledger.mu.Unlock()
			}
		}
	}
	tx.State = StateCompleted
	return nil
}
