package randomx

import "testing"

func TestSeedReproducibility(t *testing.T) {
	m1 := NewManager(1234)
	m2 := NewManager(1234)
	s1 := m1.Stream("entity:User")
	s2 := m2.Stream("entity:User")
	for i := 0; i < 100; i++ {
		a, b := s1.Int63(), s2.Int63()
		if a != b {
			t.Fatalf("streams diverged at iteration %d: %d != %d", i, a, b)
		}
	}
}

func TestDifferentStreamsDiffer(t *testing.T) {
	m := NewManager(1)
	a := m.Stream("a").Int63()
	b := m.Stream("b").Int63()
	if a == b {
		t.Fatalf("expected independent streams to differ (extremely unlikely collision)")
	}
}

func TestDistributionSampleWithinBounds(t *testing.T) {
	m := NewManager(7)
	r := m.Stream("dist")
	d := Distribution{Kind: "uniform", Min: 10, Max: 20}
	for i := 0; i < 1000; i++ {
		v := d.Sample(r)
		if v < 10 || v > 20 {
			t.Fatalf("uniform sample %v out of [10,20]", v)
		}
	}
}
