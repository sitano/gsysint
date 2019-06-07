package gsysint

import (
	"errors"
	"runtime"
	"strconv"
	"strings"
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
