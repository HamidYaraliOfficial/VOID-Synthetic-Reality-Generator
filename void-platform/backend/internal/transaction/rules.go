// Package transaction implements the Transaction Engine + Business Rules
// Engine: synthetic Order/Payment/Reservation/Inventory operations that must
// respect simple domain rules such as "if balance < amount then reject" or
// "if stock == 0 then cancel order" before they are allowed to commit.
package transaction

import "fmt"

// Operator enumerates comparisons the rule evaluator supports.
type Operator string

const (
	OpLT  Operator = "<"
	OpLTE Operator = "<="
	OpGT  Operator = ">"
	OpGTE Operator = ">="
	OpEQ  Operator = "=="
	OpNEQ Operator = "!="
)

// Condition is one "field OP value" clause. Multiple Conditions on a Rule
// are combined with logical AND.
type Condition struct {
	Field string      `json:"field"`
	Op    Operator    `json:"op"`
	Value interface{} `json:"value"`
}

// Rule is a business rule: "if all Conditions hold, ThenAction happens".
// e.g. {Conditions:[{balance, <, amount}], ThenAction: reject}
type Rule struct {
	Name       string      `json:"name"`
	Conditions []Condition `json:"conditions"`
	ThenAction string      `json:"thenAction"` // reject | approve | cancel | flag | custom
}

// Engine evaluates a rule set against a transaction's field map.
type Engine struct {
	Rules []Rule
}

func NewEngine(rules []Rule) *Engine {
	return &Engine{Rules: rules}
}

// Evaluate runs every rule against fields (in declared order) and returns the
// action of the first rule whose conditions all match, or "" if none match
// (meaning the transaction proceeds normally).
func (eng *Engine) Evaluate(fields map[string]interface{}) (matchedRule string, action string) {
	for _, rule := range eng.Rules {
		if allConditionsHold(rule.Conditions, fields) {
			return rule.Name, rule.ThenAction
		}
	}
	return "", ""
}

func allConditionsHold(conds []Condition, fields map[string]interface{}) bool {
	for _, c := range conds {
		if !conditionHolds(c, fields) {
			return false
		}
	}
	return true
}

func conditionHolds(c Condition, fields map[string]interface{}) bool {
	actual, ok := fields[c.Field]
	if !ok {
		return false
	}
	af, aok := toFloat(actual)
	vf, vok := toFloat(c.Value)
	if aok && vok {
		switch c.Op {
		case OpLT:
			return af < vf
		case OpLTE:
			return af <= vf
		case OpGT:
			return af > vf
		case OpGTE:
			return af >= vf
		case OpEQ:
			return af == vf
		case OpNEQ:
			return af != vf
		}
	}
	switch c.Op {
	case OpEQ:
		return fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", c.Value)
	case OpNEQ:
		return fmt.Sprintf("%v", actual) != fmt.Sprintf("%v", c.Value)
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
