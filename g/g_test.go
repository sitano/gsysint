package g

import (
	"testing"
)

func TestGetG(t *testing.T) {
	g := GetG()

	if g == nil {
		t.Fatalf("GetG() returned nil pointer to the g structure")
	}

	t.Log("*g =", g)
	// t.Log("g =", fmt.Sprintf("%#v", (*G)(g)))
}

func TestGetGID(t *testing.T) {
	t.Log("goroutine id = ", CurG().GoID)
}

func TestGetM(t *testing.T) {
	m := GetM()

	if m == nil {
		t.Fatalf("GetM() returned nil pointer to the m structure")
	}

	t.Log("*m =", m)
	// t.Log("m =", fmt.Sprintf("%#v", (*M)(m)))
}
