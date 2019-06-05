// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Time-related runtime and pieces of package time.

package g

// Package time knows the layout of this structure.
// If this struct changes, adjust ../time/sleep.go:/runtimeTimer.
// For GOOS=nacl, package syscall knows the layout of this structure.
// If this struct changes, adjust ../syscall/net_nacl.go:/runtimeTimer.
type Timer struct {
	TB *TimersBucket // the bucket the timer lives in
	I  int           // heap index

	// Timer wakes up at when, and then at when+period, ... (period > 0 only)
	// each time calling f(arg, now) in the timer goroutine, so f must be
	// a well-behaved function and not block.
	When   int64
	Period int64
	F      func(interface{}, uintptr)
	Arg    interface{}
	Seq    uintptr
}

//go:notinheap
type TimersBucket struct {
	lock         Mutex
	gp           *G
	created      bool
	sleeping     bool
	rescheduling bool
	sleepUntil   int64
	waitnote     Note
	t            []*Timer
}

