#include "textflag.h"

// offset of g.m, see G in runtime2.go.
#define g_m 48

TEXT ·GetG(SB),NOSPLIT,$0-8
	MOVD	g, R0
	MOVD	R0, ret+0(FP)
	RET

TEXT ·GetM(SB),NOSPLIT,$0-8
	MOVD	g_m(g), R0
	MOVD	R0, ret+0(FP)
	RET
