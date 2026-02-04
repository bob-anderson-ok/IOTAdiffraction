package main

import (
	"fmt"
	"image"
	"math/rand"
	"time"
)

func runMatrixMultiplicationTest() {
	// Test of matrix multiplication

	// Deterministic seed
	seed := int64(1)

	rng := rand.New(rand.NewSource(seed))

	numPts := 4000
	// Sizes:
	M, N, K := numPts, numPts, numPts

	// BLAS uses column-major; leading dimensions:
	lda := M
	ldb := K
	ldc := M

	A := make([]complex128, lda*K)
	B := make([]complex128, ldb*N)
	C := make([]complex128, ldc*N)

	fillComplex(rng, A)
	fillComplex(rng, B)

	alpha := complex(1.0, 0.0)
	beta := complex(0.0, 0.0)

	start := time.Now()

	Zgemm3m(Rowmajor, Notrans, Notrans, N, M, K, alpha, A, lda, B, ldb, beta, C, ldc)

	elapsed := time.Since(start)
	fmt.Printf("%d x %d complex matrix multiplication took: %v\n", numPts, numPts, elapsed)
}

// Test of fresnelCephesScalar() - visually compared against the Python version
func runFresnelCephesScalarTest() {
	xTestVals := Linspace(-5, 10, 31)
	for _, x := range xTestVals {
		S, C := fresnelCephesScalar(x)
		fmt.Printf("x=%f, S=%0.16f, C=%0.16f\n", x, S, C)
	}
	x := 40000.0
	S, C := fresnelCephesScalar(x)
	fmt.Printf("x=%f, S=%0.16f, C=%0.16f\n", x, S, C)
}

func FillGray16Random(img *image.Gray16) {
	seed := time.Now().UnixNano()

	rand := rand.New(rand.NewSource(seed))

	for y := 0; y < img.Rect.Dy(); y++ {
		row := y * img.Stride
		for x := 0; x < img.Rect.Dx(); x++ {
			// Gray16 uses big-endian per pixel: high byte, then low byte
			v := uint16(rand.Intn(65536)) // Generate values between 0 and 65535 inclusive
			i := row + 2*x
			img.Pix[i] = uint8(v >> 8)
			img.Pix[i+1] = uint8(v)
		}
	}
}

func FillGrayRandom(img *image.Gray) {
	rand.Seed(time.Now().UnixNano())

	for y := 0; y < img.Rect.Dy(); y++ {
		row := y * img.Stride
		for x := 0; x < img.Rect.Dx(); x++ {
			img.Pix[row+x] = uint8(rand.Intn(256))
		}
	}
}
