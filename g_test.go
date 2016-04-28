package gsysint

import "testing"

func TestGetG(t *testing.T) {
	g := getg()

	if g == nil {
		t.Fatalf("getg() returned nil pointer to the g structure")
	}

	t.Log("*g =", g)
}
