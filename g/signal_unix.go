// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package g

// gsignalStack saves the fields of the gsignal stack changed by
// setGsignalStack.
type GSignalStack struct {
	stack       Stack
	stackguard0 uintptr
	stackguard1 uintptr
	stktopsp    uintptr
}

