package gsysint

import "testing"

func TestGIDFromStack(t *testing.T) {
	id, err := GIDFromStackTrace()
	if err != nil {
		t.Fatal(err)
	}
	id2 := GoID()
	if id != id2 {
		t.Error("ids are different", id, id2)
	}
}
