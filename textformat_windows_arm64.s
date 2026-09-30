#include "textflag.h"

// SyscallN only supplies integer registers on Windows ARM64. Adapt its
// integer-only arguments to IDWriteFactory::CreateTextFormat's native ABI.
TEXT ·textFormatTrampolineAddr(SB),NOSPLIT,$0-8
	MOVD $textFormatTrampoline<>(SB), R0
	MOVD R0, ret+0(FP)
	RET

// Called on the system stack by SyscallN. This tail call changes no callee-save
// registers and preserves the native return address and stack pointer.
TEXT textFormatTrampoline<>(SB),NOSPLIT|NOFRAME,$0
	MOVD R0, R9 // function
	MOVD R1, R0 // this
	MOVD R2, R1 // family
	MOVD R3, R2 // collection
	MOVD R4, R3 // weight
	MOVD R5, R4 // style
	MOVD R6, R5 // stretch
	FMOVS R7, F0 // fontSize
	MOVD 0(RSP), R6 // locale
	MOVD 8(RSP), R7 // format output
	B (R9)
