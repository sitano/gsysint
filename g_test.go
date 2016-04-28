package gsysint

import "testing"

func TestGetG(t *testing.T) {
	g := GetG()

	if g == nil {
		t.Fatalf("GetG() returned nil pointer to the g structure")
	}

	t.Log("*g =", g)
}

func TestGetM(t *testing.T) {
	m := GetM()

	if m == nil {
		t.Fatalf("GetM() returned nil pointer to the m structure")
	}

	t.Log("*m =", m)
}
