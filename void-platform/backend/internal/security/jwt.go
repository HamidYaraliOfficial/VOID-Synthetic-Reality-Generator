// Package security implements API Authentication (HMAC-signed JWT, stdlib
// only), Role-Based Access Control, and an append-only Audit Log — the
// baseline security requirements from the product spec — without any
// external dependency, so the backend builds fully offline.
package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Claims is the payload of a VOID API token.
type Claims struct {
	Subject   string   `json:"sub"`
	Roles     []string `json:"roles"`
	IssuedAt  int64    `json:"iat"`
	ExpiresAt int64    `json:"exp"`
}

// TokenIssuer signs and verifies HMAC-SHA256 JWT-compatible tokens using a
// server-side secret (loaded from VOID_JWT_SECRET; see cmd/api/main.go).
type TokenIssuer struct {
	secret []byte
}

func NewTokenIssuer(secret string) *TokenIssuer {
	return &TokenIssuer{secret: []byte(secret)}
}

func b64(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// Issue creates a signed token for subject+roles valid for ttl.
func (ti *TokenIssuer) Issue(subject string, roles []string, ttl time.Duration) (string, error) {
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	headerJSON, _ := json.Marshal(header)
	now := time.Now()
	claims := Claims{
		Subject:   subject,
		Roles:     roles,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(ttl).Unix(),
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	unsigned := b64(headerJSON) + "." + b64(claimsJSON)
	mac := hmac.New(sha256.New, ti.secret)
	mac.Write([]byte(unsigned))
	sig := b64(mac.Sum(nil))
	return unsigned + "." + sig, nil
}

// Verify checks signature + expiry and returns the parsed Claims.
func (ti *TokenIssuer) Verify(token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("security: malformed token")
	}
	unsigned := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, ti.secret)
	mac.Write([]byte(unsigned))
	expectedSig := b64(mac.Sum(nil))
	if subtle.ConstantTimeCompare([]byte(expectedSig), []byte(parts[2])) != 1 {
		return nil, errors.New("security: invalid signature")
	}
	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	var claims Claims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, err
	}
	if time.Now().Unix() > claims.ExpiresAt {
		return nil, errors.New("security: token expired")
	}
	return &claims, nil
}

// --- RBAC -------------------------------------------------------------------

// Role names used across the API.
const (
	RoleAdmin   = "admin"
	RoleEngineer = "engineer"
	RoleViewer  = "viewer"
)

// Permission enumerates guarded API actions.
type Permission string

const (
	PermSimulationWrite Permission = "simulation:write"
	PermSimulationRead  Permission = "simulation:read"
	PermScenarioWrite   Permission = "scenario:write"
	PermExport          Permission = "export:run"
	PermAdmin           Permission = "admin:manage"
)

var rolePermissions = map[string]map[Permission]bool{
	RoleAdmin: {
		PermSimulationWrite: true, PermSimulationRead: true,
		PermScenarioWrite: true, PermExport: true, PermAdmin: true,
	},
	RoleEngineer: {
		PermSimulationWrite: true, PermSimulationRead: true,
		PermScenarioWrite: true, PermExport: true,
	},
	RoleViewer: {
		PermSimulationRead: true,
	},
}

// HasPermission returns true if any of the given roles grants perm.
func HasPermission(roles []string, perm Permission) bool {
	for _, role := range roles {
		if rolePermissions[role][perm] {
			return true
		}
	}
	return false
}

// RateLimiter is a simple per-key token-bucket limiter (in-memory), enough
// to protect the API server from accidental runaway clients.
type RateLimiter struct {
	buckets map[string]*bucket
	rate    float64 // tokens per second
	burst   float64
}

type bucket struct {
	tokens   float64
	lastFill time.Time
}

func NewRateLimiter(ratePerSecond, burst float64) *RateLimiter {
	return &RateLimiter{buckets: map[string]*bucket{}, rate: ratePerSecond, burst: burst}
}

func (rl *RateLimiter) Allow(key string) bool {
	now := time.Now()
	b, ok := rl.buckets[key]
	if !ok {
		b = &bucket{tokens: rl.burst, lastFill: now}
		rl.buckets[key] = b
	}
	elapsed := now.Sub(b.lastFill).Seconds()
	b.tokens += elapsed * rl.rate
	if b.tokens > rl.burst {
		b.tokens = rl.burst
	}
	b.lastFill = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// --- Audit log ---------------------------------------------------------------

// AuditEntry is one recorded security-relevant action.
type AuditEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Actor     string    `json:"actor"`
	Action    string    `json:"action"`
	Target    string    `json:"target,omitempty"`
	Detail    string    `json:"detail,omitempty"`
}

// AuditLog is an in-memory, append-only, size-bounded audit trail. For
// production use it can be tailed/forwarded to disk via WriteFunc.
type AuditLog struct {
	entries   []AuditEntry
	max       int
	WriteFunc func(AuditEntry) // optional sink, e.g. append to a file
}

func NewAuditLog(max int) *AuditLog {
	if max <= 0 {
		max = 10000
	}
	return &AuditLog{max: max}
}

func (a *AuditLog) Record(actor, action, target, detail string) {
	e := AuditEntry{Timestamp: time.Now(), Actor: actor, Action: action, Target: target, Detail: detail}
	a.entries = append(a.entries, e)
	if len(a.entries) > a.max {
		a.entries = a.entries[len(a.entries)-a.max:]
	}
	if a.WriteFunc != nil {
		a.WriteFunc(e)
	}
}

func (a *AuditLog) Recent(n int) []AuditEntry {
	if n <= 0 || n > len(a.entries) {
		n = len(a.entries)
	}
	return append([]AuditEntry{}, a.entries[len(a.entries)-n:]...)
}

// ValidateInput is a tiny helper used by API handlers to reject obviously
// unsafe identifiers (table/entity/type names) before they reach export or
// database code paths.
func ValidateInput(name string) error {
	if name == "" || len(name) > 128 {
		return fmt.Errorf("security: invalid identifier length")
	}
	for _, r := range name {
		ok := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.'
		if !ok {
			return fmt.Errorf("security: identifier contains disallowed character %q", r)
		}
	}
	return nil
}
