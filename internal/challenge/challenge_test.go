package challenge

import "testing"

func TestCatalogValidAndIDsUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range Core() {
		if err := c.Validate(); err != nil {
			t.Error(err)
		}
		if seen[c.ID] {
			t.Errorf("duplicate id %s", c.ID)
		}
		seen[c.ID] = true
		if len(c.Hints) < 2 {
			t.Errorf("%s lacks progressive hints", c.ID)
		}
		if _, err := c.Hash(); err != nil {
			t.Error(err)
		}
	}
}

func TestHashIndependentOfMapInsertion(t *testing.T) {
	a := Core()[0]
	b := a
	b.Fixtures = map[string]string{"z": "last", "a": "first"}
	a.Fixtures = map[string]string{"a": "first", "z": "last"}
	ha, _ := a.Hash()
	hb, _ := b.Hash()
	if ha != hb {
		t.Fatalf("hashes differ: %s %s", ha, hb)
	}
}

func TestRejectFixtureEscape(t *testing.T) {
	c := Core()[0]
	c.Fixtures = map[string]string{"../secret": "x"}
	if c.Validate() == nil {
		t.Fatal("expected path escape rejection")
	}
}
