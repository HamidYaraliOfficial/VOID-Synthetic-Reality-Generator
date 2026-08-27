package simulation

import (
	"testing"

	"void-platform/backend/internal/entity"
)

func testSchema() *entity.Schema {
	return &entity.Schema{
		Name: "TestUser",
		Fields: []entity.Field{
			{Name: "id", Type: entity.FieldUUID, Generator: entity.GenUUID},
			{Name: "name", Type: entity.FieldString, Generator: entity.GenName},
			{Name: "score", Type: entity.FieldInteger, Generator: entity.GenNumber,
				Params: map[string]interface{}{"min": 0.0, "max": 100.0}},
		},
	}
}

func TestUniverseGenerationIsReproducibleForSameSeed(t *testing.T) {
	u1 := NewUniverse("u1", "Test", 555)
	u2 := NewUniverse("u2", "Test", 555)
	_ = u1.AddSchema(testSchema())
	_ = u2.AddSchema(testSchema())

	e1, err := u1.SpawnEntities("TestUser", 50, nil)
	if err != nil {
		t.Fatal(err)
	}
	e2, err := u2.SpawnEntities("TestUser", 50, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(e1) != len(e2) {
		t.Fatalf("entity count mismatch: %d vs %d", len(e1), len(e2))
	}
	for i := range e1 {
		if e1[i].ID != e2[i].ID {
			t.Fatalf("entity %d ID mismatch for identical seed: %s vs %s", i, e1[i].ID, e2[i].ID)
		}
		if e1[i].Attributes["name"] != e2[i].Attributes["name"] {
			t.Fatalf("entity %d name mismatch for identical seed", i)
		}
	}
}

func TestUniverseGenerationDiffersForDifferentSeed(t *testing.T) {
	u1 := NewUniverse("u1", "Test", 1)
	u2 := NewUniverse("u2", "Test", 2)
	_ = u1.AddSchema(testSchema())
	_ = u2.AddSchema(testSchema())
	e1, _ := u1.SpawnEntities("TestUser", 10, nil)
	e2, _ := u2.SpawnEntities("TestUser", 10, nil)
	same := true
	for i := range e1 {
		if e1[i].ID != e2[i].ID {
			same = false
			break
		}
	}
	if same {
		t.Fatalf("expected different seeds to produce different entity IDs")
	}
}

func TestRelationshipAwareGeneratorLinksExistingParent(t *testing.T) {
	u := NewUniverse("u", "Test", 99)
	_ = u.AddSchema(testSchema())
	if _, err := u.SpawnEntities("TestUser", 20, nil); err != nil {
		t.Fatal(err)
	}
	orderSchema := &entity.Schema{
		Name: "TestOrder",
		Fields: []entity.Field{
			{Name: "id", Type: entity.FieldUUID, Generator: entity.GenUUID},
			{Name: "userId", Type: entity.FieldString, Generator: entity.GenDependent,
				Params: map[string]interface{}{"relatedType": "TestUser"}},
		},
	}
	_ = u.AddSchema(orderSchema)
	orders, err := u.SpawnEntities("TestOrder", 5, nil)
	if err != nil {
		t.Fatal(err)
	}
	validIDs := map[string]bool{}
	for _, e := range u.Collection("TestUser").All() {
		validIDs[e.ID] = true
	}
	for _, o := range orders {
		uid, _ := o.Attributes["userId"].(string)
		if uid == "" || !validIDs[uid] {
			t.Fatalf("order %s has invalid/missing userId %q", o.ID, uid)
		}
	}
}
