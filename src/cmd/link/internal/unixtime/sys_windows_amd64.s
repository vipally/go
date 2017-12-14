TEXT unixtime·now(SB),NOSPLIT,$0-24
	MOVQ	$_INTERRUPT_TIME, DI
loop:
	MOVL	time_hi1(DI), AX
	MOVL	time_lo(DI), BX
	MOVL	time_hi2(DI), CX
	CMPL	AX, CX
	JNE	loop
	SHLQ	$32, AX
	ORQ	BX, AX
	IMULQ	$100, AX
	SUBQ	unixtime·startNano(SB), AX
	MOVQ	AX, mono+16(FP)

	MOVQ	$_SYSTEM_TIME, DI
wall:
	MOVL	time_hi1(DI), AX
	MOVL	time_lo(DI), BX
	MOVL	time_hi2(DI), CX
	CMPL	AX, CX
	JNE	wall
	SHLQ	$32, AX
	ORQ	BX, AX
	MOVQ	$116444736000000000, DI
	SUBQ	DI, AX
	IMULQ	$100, AX

	// generated code for
	//	func f(x uint64) (uint64, uint64) { return x/1000000000, x%100000000 }
	// adapted to reduce duplication
	MOVQ	AX, CX
	MOVQ	$1360296554856532783, AX
	MULQ	CX
	ADDQ	CX, DX
	RCRQ	$1, DX
	SHRQ	$29, DX
	MOVQ	DX, sec+0(FP)
	IMULQ	$1000000000, DX
	SUBQ	DX, CX
	MOVL	CX, nsec+8(FP)
	RET