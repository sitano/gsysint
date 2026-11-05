//go:build !goexperiment.staticlockranking

package g

// lockRankStruct mirrors runtime.lockRankStruct: empty when staticlockranking is disabled (the default).
type lockRankStruct struct{}
