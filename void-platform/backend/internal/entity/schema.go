// Package entity implements the Entity Designer data model: Schemas, typed
// Fields and Relationships between Entity types, plus the runtime Collection
// that stores generated Entity instances for a running Universe.
package entity

import (
	"fmt"
	"sync"
	"time"
)

// FieldType enumerates every scalar/composite type the Entity Designer
// exposes in the UI field-type dropdown.
type FieldType string

const (
	FieldString   FieldType = "string"
	FieldInteger  FieldType = "integer"
	FieldFloat    FieldType = "float"
	FieldBoolean  FieldType = "boolean"
	FieldDateTime FieldType = "datetime"
	FieldUUID     FieldType = "uuid"
	FieldEnum     FieldType = "enum"
	FieldArray    FieldType = "array"
	FieldJSON     FieldType = "json"
	FieldBinary   FieldType = "binary"
	FieldCustom   FieldType = "custom"
)

// GeneratorKind enumerates supported per-field generator strategies.
type GeneratorKind string

const (
	GenRandom       GeneratorKind = "random"
	GenSequential   GeneratorKind = "sequential"
	GenWeighted     GeneratorKind = "weighted_random"
	GenUUID         GeneratorKind = "uuid"
	GenName         GeneratorKind = "name"
	GenEmail        GeneratorKind = "email"
	GenPhone        GeneratorKind = "phone"
	GenAddress      GeneratorKind = "address"
	GenDate         GeneratorKind = "date"
	GenTime         GeneratorKind = "time"
	GenNumber       GeneratorKind = "number"
	GenRegexPattern GeneratorKind = "pattern"
	GenDistribution GeneratorKind = "distribution"
	GenDependent    GeneratorKind = "dependent"
	GenDerived      GeneratorKind = "derived"
	GenCustom       GeneratorKind = "custom_function"
)

// Field describes one attribute of an Entity Schema.
type Field struct {
	Name        string                 `json:"name"`
	Type        FieldType              `json:"type"`
	Generator   GeneratorKind          `json:"generator"`
	Required    bool                   `json:"required"`
	Unique      bool                   `json:"unique"`
	EnumValues  []string               `json:"enumValues,omitempty"`
	Params      map[string]interface{} `json:"params,omitempty"` // generator-specific config
	DependsOn   string                 `json:"dependsOn,omitempty"`
	Expression  string                 `json:"expression,omitempty"` // for derived fields
}

// RelationshipKind enumerates supported entity-to-entity relationships.
type RelationshipKind string

const (
	OneToOne    RelationshipKind = "one_to_one"
	OneToMany   RelationshipKind = "one_to_many"
	ManyToMany  RelationshipKind = "many_to_many"
	Hierarchical RelationshipKind = "hierarchical"
)

// Relationship links two entity types together, e.g. Customer -1:N-> Order.
type Relationship struct {
	Name       string           `json:"name"`
	FromEntity string           `json:"fromEntity"`
	ToEntity   string           `json:"toEntity"`
	Kind       RelationshipKind `json:"kind"`
	MinCount   int              `json:"minCount"`
	MaxCount   int              `json:"maxCount"`
	ForeignKey string           `json:"foreignKey"` // field on "to" pointing back to "from"
}

// Schema is the blueprint for one Entity type (User, Order, Server, ...).
type Schema struct {
	Name          string         `json:"name"`
	Description   string         `json:"description,omitempty"`
	Fields        []Field        `json:"fields"`
	Relationships []Relationship `json:"relationships,omitempty"`
	States        []string       `json:"states,omitempty"`  // lifecycle states
	InitialState  string         `json:"initialState,omitempty"`
}

// Validate performs structural sanity checks on a schema before it can be
// used to generate data (unique field names, valid enum config, etc).
func (s *Schema) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("schema: name is required")
	}
	seen := map[string]bool{}
	for _, f := range s.Fields {
		if f.Name == "" {
			return fmt.Errorf("schema %s: field with empty name", s.Name)
		}
		if seen[f.Name] {
			return fmt.Errorf("schema %s: duplicate field %q", s.Name, f.Name)
		}
		seen[f.Name] = true
		if f.Type == FieldEnum && len(f.EnumValues) == 0 {
			return fmt.Errorf("schema %s: enum field %q needs enumValues", s.Name, f.Name)
		}
	}
	return nil
}

// Entity is one concrete, generated instance of a Schema.
type Entity struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Attributes map[string]interface{} `json:"attributes"`
	State      string                 `json:"state,omitempty"`
	CreatedAt  time.Time              `json:"createdAt"`
	UpdatedAt  time.Time              `json:"updatedAt"`
	Links      map[string][]string    `json:"links,omitempty"` // relationshipName -> related entity IDs
}

// Collection is a concurrency-safe store of all Entity instances of one type.
type Collection struct {
	mu       sync.RWMutex
	TypeName string
	items    map[string]*Entity
	order    []string // preserves insertion order for deterministic export
}

func NewCollection(typeName string) *Collection {
	return &Collection{TypeName: typeName, items: make(map[string]*Entity)}
}

func (c *Collection) Add(e *Entity) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.items[e.ID]; !exists {
		c.order = append(c.order, e.ID)
	}
	c.items[e.ID] = e
}

func (c *Collection) Get(id string) (*Entity, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.items[id]
	return e, ok
}

func (c *Collection) Delete(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, id)
}

func (c *Collection) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// All returns a stable-ordered snapshot slice of every entity currently held.
func (c *Collection) All() []*Entity {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]*Entity, 0, len(c.order))
	for _, id := range c.order {
		if e, ok := c.items[id]; ok {
			out = append(out, e)
		}
	}
	return out
}

// Random returns a uniformly random existing entity, used heavily by the
// relationship-aware generator to attach e.g. an Order to an existing
// Customer instead of inventing an orphaned foreign key.
func (c *Collection) Random(intn func(int) int) (*Entity, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.order) == 0 {
		return nil, false
	}
	id := c.order[intn(len(c.order))]
	e, ok := c.items[id]
	return e, ok
}
