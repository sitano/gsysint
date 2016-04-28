package gsysint

import "testing"

func TestGetG(t *testing.T) {
	g := getg()

	if g == nil {
		t.Fatalf("getg() returned nil pointer to the g structure")
	}

	t.Log("*g =", g)
}

func TestGetM(t *testing.T) {
	m := getm()

	if m == nil {
		t.Fatalf("getm() returned nil pointer to the m structure")
	}

	t.Log("*m =", m)
}
