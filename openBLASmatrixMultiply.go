package main

/*
#include "cblas.h"
#include <stdlib.h>
#cgo LDFLAGS: -L. -l:libopenblas.a     -l:libgomp.a -l:libssp.a  -l:libucrt.a
*/
import "C"
import "unsafe"

type Order int32
type Transpose int32

const (
	Rowmajor Order = C.CblasRowMajor
	//Colmajor Order = C.CblasColMajor
)

const (
	Notrans Transpose = C.CblasNoTrans
	//Trans       Transpose = C.CblasTrans
	//Conjtrans   Transpose = C.CblasConjTrans
	//Conjnotrans Transpose = C.CblasConjNoTrans
)

//func GetNumThreads() int32 {
//	return int32(C.openblas_get_num_threads())
//}

// Zgemm3m performs matrix-matrix multiplication with complex double-precision matrix A and matrix B and stores the result in matrix C.
func Zgemm3m(order Order, transA, transB Transpose, m, n, k int,
	alpha complex128, a []complex128, lda int, b []complex128, ldb int, beta complex128, c []complex128, ldc int) {
	C.cblas_zgemm3m(C.enum_CBLAS_ORDER(order), C.enum_CBLAS_TRANSPOSE(transA), C.enum_CBLAS_TRANSPOSE(transB), C.blasint(m), C.blasint(n), C.blasint(k), unsafe.Pointer(&alpha), unsafe.Pointer(&a[0]), C.blasint(lda), unsafe.Pointer(&b[0]), C.blasint(ldb), unsafe.Pointer(&beta), unsafe.Pointer(&c[0]), C.blasint(ldc))
}
