// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Time-related runtime and pieces of package time.

package g

import (
	"unsafe"
)

// Timer is the runtime timer of go1.27.
// Package time knows the layout of the user visible part of this structure.
type Timer struct {
	mu Mutex

	astate uint8 // atomic copy of state bits at last unlock
	state  uint8 // state bits
	isChan bool  // timer has a channel; immutable; can be read without lock
	isFake bool  // timer is using fake time; immutable; can be read without lock

	blocked uint32 // number of goroutines blocked on timer's channel
	rand    uint32 // randomizes order of timers at same instant; only set when isFake

	// Timer wakes up at when, and then at when+period, ... (period > 0 only)
	// each time calling f(arg, seq, delay) in the timer goroutine, so f must be
	// a well-behaved function and not block.
	When   int64
	Period int64
	F      func(arg any, seq uintptr, delay int64)
	Arg    any
	Seq    uintptr

	ts unsafe.Pointer // *timers

	sendLock Mutex

	isSending int32
}
