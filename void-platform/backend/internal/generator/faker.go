// Package generator turns an entity.Schema plus a seeded randomx.Manager
// into realistic-looking, fully synthetic Entity instances: no real personal
// data ever flows through this package, only statistically shaped fakes.
package generator

import (
	"fmt"
	"math/rand"
	"regexp/syntax"
	"strings"
	"time"
)

var firstNames = []string{
	"Alex", "Sara", "Milad", "Nina", "Kian", "Leila", "Arman", "Dara", "Yara", "Sam",
	"Wei", "Mei", "Jian", "Ling", "Hao", "Omid", "Parisa", "Reza", "Tara", "Ehsan",
	"Noah", "Emma", "Liam", "Olivia", "Ava", "Mason", "Isla", "Ethan", "Zoe", "Kai",
}
var lastNames = []string{
	"Karimi", "Zhang", "Smith", "Ahmadi", "Chen", "Johnson", "Rahimi", "Wang", "Brown", "Li",
	"Hosseini", "Liu", "Garcia", "Moradi", "Huang", "Davis", "Sadeghi", "Yang", "Miller", "Wu",
}
var streetNames = []string{"Maple", "Oak", "Pine", "Elm", "Cedar", "Sunset", "Azadi", "Valiasr", "Nanjing", "Zhongshan"}
var cities = []string{"Tehran", "Shanghai", "Berlin", "Frankfurt", "Beijing", "Isfahan", "Shenzhen", "Munich", "Guangzhou", "Mashhad"}
var domains = []string{"example.com", "voidmail.test", "synthetic.dev", "mailbox.test"}
var companyWords = []string{"Nova", "Vertex", "Quantum", "Nimbus", "Orbit", "Delta", "Prime", "Zenith", "Pulse", "Vector"}

// FullName returns a synthetic "First Last" name.
func FullName(r *rand.Rand) string {
	return firstNames[r.Intn(len(firstNames))] + " " + lastNames[r.Intn(len(lastNames))]
}

// Email returns a synthetic, clearly-fake email address derived from a name.
func Email(r *rand.Rand, name string) string {
	slug := strings.ToLower(strings.ReplaceAll(name, " ", "."))
	if slug == "" {
		slug = fmt.Sprintf("user%d", r.Intn(999999))
	}
	return fmt.Sprintf("%s.%d@%s", slug, r.Intn(9999), domains[r.Intn(len(domains))])
}

// Phone returns a synthetic phone number using an obviously non-routable
// prefix pattern so it can never collide with a real subscriber number.
func Phone(r *rand.Rand) string {
	return fmt.Sprintf("+1-555-%03d-%04d", r.Intn(1000), r.Intn(10000))
}

// Address returns a synthetic street address.
func Address(r *rand.Rand) string {
	return fmt.Sprintf("%d %s St, %s", 1+r.Intn(9998), streetNames[r.Intn(len(streetNames))], cities[r.Intn(len(cities))])
}

// CompanyName returns a synthetic company name.
func CompanyName(r *rand.Rand) string {
	return companyWords[r.Intn(len(companyWords))] + " " + companyWords[r.Intn(len(companyWords))] + " Inc."
}

// UUIDv4 returns a random RFC-4122 v4 UUID string sourced from the given
// deterministic stream, so universes stay reproducible for a fixed seed.
func UUIDv4(r *rand.Rand) string {
	b := make([]byte, 16)
	r.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// RandomDate returns a random time between start and end.
func RandomDate(r *rand.Rand, start, end time.Time) time.Time {
	if end.Before(start) {
		start, end = end, start
	}
	delta := end.Sub(start)
	if delta <= 0 {
		return start
	}
	return start.Add(time.Duration(r.Int63n(int64(delta))))
}

// FromPattern generates a string that matches a (restricted, non-catastrophic)
// regular expression pattern, used by the "pattern" field generator so users
// can spec things like order codes: ORD-[0-9]{6}.
func FromPattern(r *rand.Rand, pattern string) (string, error) {
	re, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	generateFromRegexAST(r, re, &sb, 0)
	return sb.String(), nil
}

func generateFromRegexAST(r *rand.Rand, re *syntax.Regexp, sb *strings.Builder, depth int) {
	if depth > 50 {
		return
	}
	switch re.Op {
	case syntax.OpLiteral:
		for _, ru := range re.Rune {
			sb.WriteRune(ru)
		}
	case syntax.OpCharClass:
		if len(re.Rune) >= 2 {
			lo, hi := re.Rune[0], re.Rune[1]
			sb.WriteRune(lo + rune(r.Intn(int(hi-lo+1))))
		}
	case syntax.OpAnyChar, syntax.OpAnyCharNotNL:
		sb.WriteRune(rune('a' + r.Intn(26)))
	case syntax.OpConcat:
		for _, sub := range re.Sub {
			generateFromRegexAST(r, sub, sb, depth+1)
		}
	case syntax.OpCapture:
		if len(re.Sub) > 0 {
			generateFromRegexAST(r, re.Sub[0], sb, depth+1)
		}
	case syntax.OpStar:
		n := r.Intn(4)
		for i := 0; i < n; i++ {
			if len(re.Sub) > 0 {
				generateFromRegexAST(r, re.Sub[0], sb, depth+1)
			}
		}
	case syntax.OpPlus:
		n := 1 + r.Intn(4)
		for i := 0; i < n; i++ {
			if len(re.Sub) > 0 {
				generateFromRegexAST(r, re.Sub[0], sb, depth+1)
			}
		}
	case syntax.OpRepeat:
		lo := re.Min
		hi := re.Max
		if hi < lo || hi < 0 {
			hi = lo + 2
		}
		n := lo
		if hi > lo {
			n = lo + r.Intn(hi-lo+1)
		}
		for i := 0; i < n; i++ {
			if len(re.Sub) > 0 {
				generateFromRegexAST(r, re.Sub[0], sb, depth+1)
			}
		}
	case syntax.OpAlternate:
		if len(re.Sub) > 0 {
			generateFromRegexAST(r, re.Sub[r.Intn(len(re.Sub))], sb, depth+1)
		}
	case syntax.OpQuest:
		if r.Intn(2) == 0 && len(re.Sub) > 0 {
			generateFromRegexAST(r, re.Sub[0], sb, depth+1)
		}
	}
}
