//go:build goexperiment.staticlockranking

package g

// lockRankStruct mirrors runtime.lockRankStruct: embedded in Mutex.
type lockRankStruct struct {
	// static lock ranking of the lock
	rank int64 // runtime.lockRank
}
