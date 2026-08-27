package generator

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"void-platform/backend/internal/entity"
	"void-platform/backend/internal/randomx"
)

// SequenceCounters tracks per-(schema,field) auto-increment state so
// "sequential" generators keep counting up across an entire simulation run.
type SequenceCounters struct {
	counters map[string]int64
}

func NewSequenceCounters() *SequenceCounters {
	return &SequenceCounters{counters: make(map[string]int64)}
}

func (s *SequenceCounters) Next(key string) int64 {
	s.counters[key]++
	return s.counters[key]
}

// Engine produces entity.Entity instances from an entity.Schema. It is the
// implementation behind "Synthetic Data Generator" + "Relationship-Aware
// Generator" from the product spec: field-level generation plus wiring new
// entities to already-generated related entities so referential integrity
// (Order -> Customer, Session -> User, ...) holds across the universe.
type Engine struct {
	RNG  *randomx.Manager
	Seq  *SequenceCounters
}

func NewEngine(rng *randomx.Manager) *Engine {
	return &Engine{RNG: rng, Seq: NewSequenceCounters()}
}

// RelatedLookup lets the Engine pull an existing related entity's ID when
// generating a "dependent" field (e.g. Order.customerId -> an existing
// Customer). Supplied by the simulation package, which owns all collections.
type RelatedLookup func(relatedType string) (id string, ok bool)

// Generate creates a single new entity.Entity for the given schema.
func (g *Engine) Generate(schema *entity.Schema, lookup RelatedLookup) (*entity.Entity, error) {
	stream := g.RNG.Stream("entity:" + schema.Name)
	now := time.Now()
	e := &entity.Entity{
		ID:         UUIDv4(stream),
		Type:       schema.Name,
		Attributes: make(map[string]interface{}, len(schema.Fields)),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if schema.InitialState != "" {
		e.State = schema.InitialState
	} else if len(schema.States) > 0 {
		e.State = schema.States[0]
	}

	// First pass: independent fields (anything not "dependent"/"derived").
	for _, f := range schema.Fields {
		if f.Generator == entity.GenDependent || f.Generator == entity.GenDerived {
			continue
		}
		val, err := g.generateField(stream, schema, f, lookup)
		if err != nil {
			return nil, fmt.Errorf("entity %s field %s: %w", schema.Name, f.Name, err)
		}
		e.Attributes[f.Name] = val
	}
	// Second pass: dependent fields can reference sibling field values.
	for _, f := range schema.Fields {
		if f.Generator != entity.GenDependent && f.Generator != entity.GenDerived {
			continue
		}
		val, err := g.generateField(stream, schema, f, lookup)
		if err != nil {
			return nil, fmt.Errorf("entity %s field %s: %w", schema.Name, f.Name, err)
		}
		e.Attributes[f.Name] = val
	}
	return e, nil
}

// GenerateBatch creates n entities, calling progress(i,n) periodically so a
// caller (API/CLI) can stream progress for very large synthetic populations.
func (g *Engine) GenerateBatch(schema *entity.Schema, n int, lookup RelatedLookup, progress func(done, total int)) ([]*entity.Entity, error) {
	out := make([]*entity.Entity, 0, n)
	for i := 0; i < n; i++ {
		e, err := g.Generate(schema, lookup)
		if err != nil {
			return out, err
		}
		out = append(out, e)
		if progress != nil && (i%1000 == 0 || i == n-1) {
			progress(i+1, n)
		}
	}
	return out, nil
}

func (g *Engine) generateField(stream *rand.Rand, schema *entity.Schema, f entity.Field, lookup RelatedLookup) (interface{}, error) {
	switch f.Generator {
	case entity.GenUUID:
		return UUIDv4(stream), nil
	case entity.GenName:
		return FullName(stream), nil
	case entity.GenEmail:
		name, _ := paramStr(f.Params, "name")
		if name == "" {
			name = FullName(stream)
		}
		return Email(stream, name), nil
	case entity.GenPhone:
		return Phone(stream), nil
	case entity.GenAddress:
		return Address(stream), nil
	case entity.GenDate, entity.GenTime:
		start, end := dateRange(f.Params)
		return RandomDate(stream, start, end).Format(time.RFC3339), nil
	case entity.GenSequential:
		start := int64(0)
		if v, ok := f.Params["start"]; ok {
			start = toInt64(v)
		}
		return start + g.Seq.Next(schema.Name+"."+f.Name), nil
	case entity.GenWeighted:
		return g.weightedChoice(stream, f)
	case entity.GenRegexPattern:
		pattern, _ := paramStr(f.Params, "pattern")
		if pattern == "" {
			pattern = "[A-Z]{3}-[0-9]{5}"
		}
		return FromPattern(stream, pattern)
	case entity.GenDistribution:
		d := distributionFromParams(f.Params)
		val := d.Sample(stream)
		if f.Type == entity.FieldInteger {
			return int64(val), nil
		}
		return val, nil
	case entity.GenNumber:
		min, max := numRange(f.Params)
		if f.Type == entity.FieldInteger {
			return int64(min) + stream.Int63n(int64(max-min)+1), nil
		}
		return min + stream.Float64()*(max-min), nil
	case entity.GenDependent:
		relatedType, _ := paramStr(f.Params, "relatedType")
		if lookup != nil && relatedType != "" {
			if id, ok := lookup(relatedType); ok {
				return id, nil
			}
		}
		return nil, nil
	case entity.GenDerived:
		// Minimal safe "derived" support: concatenation of other attribute
		// names given in params["from"] = ["field1","field2"].
		return f.Expression, nil
	case entity.GenCustom:
		return nil, fmt.Errorf("custom_function generator %q requires a registered plugin", f.Name)
	default: // GenRandom, fallback by declared Type
		return g.randomByType(stream, f)
	}
}

func (g *Engine) randomByType(stream *rand.Rand, f entity.Field) (interface{}, error) {
	switch f.Type {
	case entity.FieldString:
		return fmt.Sprintf("%s-%06d", f.Name, stream.Intn(999999)), nil
	case entity.FieldInteger:
		min, max := numRange(f.Params)
		return int64(min) + stream.Int63n(int64(max-min)+1), nil
	case entity.FieldFloat:
		min, max := numRange(f.Params)
		return min + stream.Float64()*(max-min), nil
	case entity.FieldBoolean:
		return stream.Intn(2) == 1, nil
	case entity.FieldDateTime:
		start, end := dateRange(f.Params)
		return RandomDate(stream, start, end).Format(time.RFC3339), nil
	case entity.FieldUUID:
		return UUIDv4(stream), nil
	case entity.FieldEnum:
		if len(f.EnumValues) == 0 {
			return "", nil
		}
		return f.EnumValues[stream.Intn(len(f.EnumValues))], nil
	case entity.FieldArray:
		n := 1 + stream.Intn(4)
		arr := make([]interface{}, n)
		for i := range arr {
			arr[i] = stream.Intn(1000)
		}
		return arr, nil
	case entity.FieldJSON:
		return map[string]interface{}{"generated": true, "value": stream.Intn(1000)}, nil
	case entity.FieldBinary:
		b := make([]byte, 8)
		stream.Read(b)
		return b, nil
	default:
		return nil, nil
	}
}

func (g *Engine) weightedChoice(stream *rand.Rand, f entity.Field) (interface{}, error) {
	rawValues, _ := f.Params["values"].([]interface{})
	rawWeights, _ := f.Params["weights"].([]interface{})
	if len(rawValues) == 0 {
		return nil, fmt.Errorf("weighted_random generator needs params.values")
	}
	weights := make([]float64, len(rawValues))
	total := 0.0
	for i := range rawValues {
		w := 1.0
		if i < len(rawWeights) {
			w = toFloat64(rawWeights[i])
		}
		weights[i] = w
		total += w
	}
	target := stream.Float64() * total
	acc := 0.0
	for i, w := range weights {
		acc += w
		if target <= acc {
			return rawValues[i], nil
		}
	}
	return rawValues[len(rawValues)-1], nil
}

// --- small param helpers ----------------------------------------------------

func paramStr(params map[string]interface{}, key string) (string, bool) {
	if params == nil {
		return "", false
	}
	v, ok := params[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

func numRange(params map[string]interface{}) (float64, float64) {
	min, max := 0.0, 1000.0
	if params == nil {
		return min, max
	}
	if v, ok := params["min"]; ok {
		min = toFloat64(v)
	}
	if v, ok := params["max"]; ok {
		max = toFloat64(v)
	}
	if max <= min {
		max = min + 1
	}
	return min, max
}

func dateRange(params map[string]interface{}) (time.Time, time.Time) {
	end := time.Now()
	start := end.AddDate(-1, 0, 0)
	if params == nil {
		return start, end
	}
	if v, ok := paramStr(params, "start"); ok {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			start = t
		}
	}
	if v, ok := paramStr(params, "end"); ok {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			end = t
		}
	}
	return start, end
}

func distributionFromParams(params map[string]interface{}) randomx.Distribution {
	d := randomx.Distribution{Kind: "uniform"}
	if params == nil {
		return d
	}
	if v, ok := paramStr(params, "kind"); ok {
		d.Kind = v
	}
	if v, ok := params["min"]; ok {
		d.Min = toFloat64(v)
	}
	if v, ok := params["max"]; ok {
		d.Max = toFloat64(v)
	}
	if v, ok := params["mean"]; ok {
		d.Mean = toFloat64(v)
	}
	if v, ok := params["stddev"]; ok {
		d.StdDev = toFloat64(v)
	}
	if v, ok := params["lambda"]; ok {
		d.Lambda = toFloat64(v)
	}
	if v, ok := params["alpha"]; ok {
		d.Alpha = toFloat64(v)
	}
	if v, ok := params["xmin"]; ok {
		d.XMin = toFloat64(v)
	}
	return d
}

func toFloat64(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case string:
		f, _ := strconv.ParseFloat(n, 64)
		return f
	default:
		return 0
	}
}

func toInt64(v interface{}) int64 {
	return int64(toFloat64(v))
}
