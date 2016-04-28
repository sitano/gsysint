package gsysint

type SigSet struct{}

type mOS struct {
	waitsema uintptr // semaphore for parking on locks
}