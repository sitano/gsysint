//go:build !darwin && !linux

package g

// SigSet layout is unknown here, so M fields after SigMask are not reliable.
type SigSet struct{}
