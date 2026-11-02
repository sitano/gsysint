// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package trace mirrors the parts of the runtime execution tracer
// (runtime/traceruntime.go) that gopark callers need.
package trace

// BlockReason is runtime.traceBlockReason: why a goroutine blocks,
// as reported to the execution tracer.
type BlockReason uint8

const (
	BlockGeneric BlockReason = iota
	BlockForever
	BlockNet
	BlockSelect
	BlockCondWait
	BlockSync
	BlockChanSend
	BlockChanRecv
	BlockGCMarkAssist
	BlockGCSweep
	BlockSystemGoroutine
	BlockPreempted
	BlockDebugCall
	BlockUntilGCEnds
	BlockSleep
	BlockGCWeakToStrongWait
	BlockSynctest
)
