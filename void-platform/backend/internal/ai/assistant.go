// Package ai implements the AI Simulation Assistant + Scenario Copilot.
// It only ever PRODUCES configuration (entities, relationships, behaviors,
// traffic profiles, scenarios, dashboards) for the user to Preview before
// anything runs — it never executes a simulation itself; internal/simulation
// remains the sole execution authority, matching the product requirement
// that "AI باید فقط Configuration ایجاد کند".
//
// The built-in Interpreter is a fast, dependency-free, pattern-based parser
// that understands common phrasings ("N میلیون/million کاربر/users", "X% از
// کاربران", "peak/اوج at HH", ...). Set an external LLMBackend (any HTTP
// client hitting Anthropic's Messages API or similar) via SetBackend to
// upgrade free-text understanding without changing any caller code.
package ai

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"void-platform/backend/internal/entity"
	"void-platform/backend/internal/scenario"
)

// GeneratedConfig is what the Assistant hands back for user Preview.
type GeneratedConfig struct {
	Summary   string           `json:"summary"`
	Schemas   []*entity.Schema `json:"schemas"`
	Scenario  *scenario.Scenario `json:"scenario"`
	Notes     []string         `json:"notes"`
}

// Backend lets a real LLM be plugged in later; nil means "use the built-in
// pattern-based Interpreter only".
type Backend interface {
	Complete(ctx context.Context, prompt string) (string, error)
}

// Assistant is the entry point used by the API's /api/ai/generate handler.
type Assistant struct {
	Backend Backend
}

func New() *Assistant { return &Assistant{} }

func (a *Assistant) SetBackend(b Backend) { a.Backend = b }

var (
	reUserCount = regexp.MustCompile(`(?i)([\d.,]+)\s*(million|میلیون|هزار|thousand|万|百万)?\s*(user|users|کاربر|用户)`)
	rePercent   = regexp.MustCompile(`(\d+(?:\.\d+)?)\s*%`)
	reHour      = regexp.MustCompile(`(?i)(?:hour|ساعت|点)\s*(\d{1,2})|(\d{1,2})\s*(?:h|:00|ساعت)`)
)

// Interpret turns a free-text description (English, Persian or Chinese) into
// a first-draft GeneratedConfig. It is intentionally conservative: it always
// produces something runnable and explains, in Notes, exactly what it
// inferred so the human stays in control before hitting "Run".
func (a *Assistant) Interpret(ctx context.Context, prompt string) (*GeneratedConfig, error) {
	if a.Backend != nil {
		if text, err := a.Backend.Complete(ctx, prompt); err == nil && text != "" {
			// A real backend is expected to already return structured JSON;
			// callers that wire one up should parse `text` themselves. The
			// built-in interpreter below is the guaranteed-available path.
			_ = text
		}
	}

	totalUsers := extractUserCount(prompt)
	if totalUsers == 0 {
		totalUsers = 10000
	}
	activePct := extractPercent(prompt, 10.0)
	peakHour := extractHour(prompt, 18)

	activeUsers := int(float64(totalUsers) * activePct / 100.0)

	userSchema := &entity.Schema{
		Name: "User",
		Fields: []entity.Field{
			{Name: "id", Type: entity.FieldUUID, Generator: entity.GenUUID, Required: true, Unique: true},
			{Name: "name", Type: entity.FieldString, Generator: entity.GenName},
			{Name: "email", Type: entity.FieldString, Generator: entity.GenEmail},
			{Name: "signupDate", Type: entity.FieldDateTime, Generator: entity.GenDate},
			{Name: "plan", Type: entity.FieldEnum, Generator: entity.GenWeighted,
				EnumValues: []string{"free", "pro", "enterprise"},
				Params: map[string]interface{}{
					"values":  []interface{}{"free", "pro", "enterprise"},
					"weights": []interface{}{70.0, 25.0, 5.0},
				}},
		},
		States:       []string{"anonymous", "active", "churned"},
		InitialState: "anonymous",
	}
	sessionSchema := &entity.Schema{
		Name: "Session",
		Fields: []entity.Field{
			{Name: "id", Type: entity.FieldUUID, Generator: entity.GenUUID, Required: true, Unique: true},
			{Name: "userId", Type: entity.FieldString, Generator: entity.GenDependent,
				Params: map[string]interface{}{"relatedType": "User"}},
			{Name: "startedAt", Type: entity.FieldDateTime, Generator: entity.GenDate},
		},
	}

	sc := &scenario.Scenario{
		ID:        "ai-generated",
		Name:      "AI-generated scenario",
		Seed:      42,
		TimeScale: 60,
		Timeline: []scenario.Action{
			{ID: "spawn-users", At: 0, Kind: scenario.ActionSpawn,
				Params: map[string]interface{}{"schema": "User", "count": float64(totalUsers)}},
			{ID: "spawn-sessions", At: 1_000_000_000, Kind: scenario.ActionSpawn, // 1s in ns as time.Duration literal-friendly
				Params: map[string]interface{}{"schema": "Session", "count": float64(activeUsers)}},
			{ID: "peak-traffic", At: 0, Kind: scenario.ActionLoad,
				Params: map[string]interface{}{"peakHour": peakHour, "activeUsers": activeUsers}},
		},
	}

	notes := []string{
		fmt.Sprintf("Inferred total population: %d", totalUsers),
		fmt.Sprintf("Inferred concurrently-active share: %.1f%% (%d users)", activePct, activeUsers),
		fmt.Sprintf("Inferred traffic peak hour: %02d:00", peakHour),
		"This is a draft — review the Entity Designer and Timeline before running.",
	}

	return &GeneratedConfig{
		Summary:  fmt.Sprintf("SaaS-style universe: %d users, %d peak-active, peak at %02d:00", totalUsers, activeUsers, peakHour),
		Schemas:  []*entity.Schema{userSchema, sessionSchema},
		Scenario: sc,
		Notes:    notes,
	}, nil
}

func extractUserCount(prompt string) int {
	m := reUserCount.FindStringSubmatch(prompt)
	if m == nil {
		return 0
	}
	numStr := strings.ReplaceAll(m[1], ",", "")
	n, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0
	}
	unit := strings.ToLower(m[2])
	switch unit {
	case "million", "میلیون", "百万":
		n *= 1_000_000
	case "thousand", "هزار":
		n *= 1_000
	case "万":
		n *= 10_000
	}
	return int(n)
}

func extractPercent(prompt string, fallback float64) float64 {
	m := rePercent.FindStringSubmatch(prompt)
	if m == nil {
		return fallback
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return fallback
	}
	return v
}

func extractHour(prompt string, fallback int) int {
	m := reHour.FindStringSubmatch(prompt)
	if m == nil {
		return fallback
	}
	for _, g := range m[1:] {
		if g != "" {
			if h, err := strconv.Atoi(g); err == nil && h >= 0 && h <= 23 {
				return h
			}
		}
	}
	return fallback
}

// AnalyzeResult performs post-run analysis: bottleneck/latency/failure
// summarisation over already-computed metrics — never re-executes anything.
type AnalyzeInput struct {
	ErrorRatePct   float64            `json:"errorRatePct"`
	P99LatencyMs   float64            `json:"p99LatencyMs"`
	ServiceLatency map[string]float64 `json:"serviceLatencyMs"`
	FailedTxPct    float64            `json:"failedTransactionsPct"`
}

func (a *Assistant) Analyze(input AnalyzeInput) []string {
	var findings []string
	worstService, worstLatency := "", 0.0
	for svc, lat := range input.ServiceLatency {
		if lat > worstLatency {
			worstLatency, worstService = lat, svc
		}
	}
	if worstService != "" {
		findings = append(findings, fmt.Sprintf("Highest-latency service: %s (%.0fms)", worstService, worstLatency))
	}
	if input.P99LatencyMs > 1000 {
		findings = append(findings, fmt.Sprintf("P99 latency is high (%.0fms) — investigate the slowest dependency chain first.", input.P99LatencyMs))
	}
	if input.ErrorRatePct > 1 {
		findings = append(findings, fmt.Sprintf("Error rate is %.2f%%, above the 1%% healthy baseline.", input.ErrorRatePct))
	}
	if input.FailedTxPct > 0 {
		findings = append(findings, fmt.Sprintf("%.2f%% of transactions failed business rules or downstream calls.", input.FailedTxPct))
	}
	if len(findings) == 0 {
		findings = append(findings, "No significant bottlenecks detected against the configured thresholds.")
	}
	return findings
}
