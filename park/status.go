package park

import "github.com/sitano/gsysint/runtime/g"

// IsParked reports whether gp is in _Gwaiting, i.e. parked by gopark for any
// reason (this package, a channel, a sync primitive, a timer...).
//
// It is a snapshot: gp may be readied right after it returns. It is safe to act
// on only when the caller is the sole party that can ready gp, which is exactly
// the GoReady contract: GoReady(gp) throws "bad g->status in ready" unless gp is
// _Gwaiting. Spinning on IsParked is an alternative to the "take the lock that
// GoParkUnlock releases" handshake.
//
// gp stays a valid pointer forever: the runtime never frees g structs, it only
// puts dead ones on a free list for reuse. A dead or reused g reports its new state.
func IsParked(gp *g.G) bool {
	return gp.Status() == g.Gwaiting
}
