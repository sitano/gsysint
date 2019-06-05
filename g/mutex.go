package g

import (
	_ "unsafe"
)

//go:linkname Lock runtime.lock
func Lock(l *Mutex)

//go:linkname Unlock runtime.unlock
func Unlock(l *Mutex)
