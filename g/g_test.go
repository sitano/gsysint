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

// Layout check against the live runtime: g.goid must match runtime's, and m.curg must point back to g.
func TestLayout(t *testing.T) {
	gp := CurG()
	if gp.M != CurM() {
		t.Error("g.m != GetM()")
	}
	if gp.M.CurG != gp {
		t.Error("m.curg != g")
	}
	if gp.M.G0 == nil || gp.M.G0.M != gp.M {
		t.Error("m.g0 mismatch")
	}
	if gp.Stack.lo == 0 || gp.Stack.hi <= gp.Stack.lo {
		t.Error("bad stack bounds", gp.Stack)
	}
}
