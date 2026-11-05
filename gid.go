package gsysint

import (
	"errors"
	"runtime"
	"strconv"
	"strings"

	"github.com/sitano/gsysint/runtime/g"
)

var ErrWrongFormat = errors.New("wrong stack format")

// GIDFromStackTrace returns id of invoker goroutine.
func GIDFromStackTrace() (uint64, error) {
	buf := make([]byte, 64)
	sz := runtime.Stack(buf, false)
	sp := strings.Split(string(buf[:sz]), " ")
	if len(sp) < 2 {
		return 0, ErrWrongFormat
	}
	id, err := strconv.ParseUint(sp[1], 10, 64)
	if err != nil {
		return 0, err
	}
	if id < 1 {
		return 0, ErrWrongFormat
	}
	return id, nil
}

// GoID returns id of invoker goroutine, read from its runtime g struct.
func GoID() uint64 { return g.CurG().GoID }
