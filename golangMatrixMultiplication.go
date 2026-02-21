package main

import (
	"fmt"
	"runtime"
	"sync"
)

// MatMulSquareComplex multiplies A*B into C for n×n complex128 matrices,
// all stored as flattened row-major slices.
//   - A, B: length n*n
//   - returns C: length n*n
//
// Implementation: blocked (tiled) GEMM with a worker pool over C-tiles.
func MatMulSquareComplex(A, B []complex128, n int) ([]complex128, error) {
	if n <= 0 {
		return nil, fmt.Errorf("n must be > 0")
	}
	if len(A) != n*n || len(B) != n*n {
		return nil, fmt.Errorf("A and B must have length n*n")
	}

	C := make([]complex128, n*n)

	// Tune these if you benchmark:
	// - block controls cache behavior
	// - workers should usually be <= GOMAXPROCS
	const block = 32
	workers := runtime.GOMAXPROCS(0)

	type tile struct{ ii, jj int } // tile origin in C
	tasks := make(chan tile, workers*2)

	var wg sync.WaitGroup
	wg.Add(workers)

	worker := func() {
		defer wg.Done()

		// Local aliases for speed
		a := A
		b := B
		c := C

		for t := range tasks {
			ii, jj := t.ii, t.jj
			iMax := min(ii+block, n)
			jMax := min(jj+block, n)

			// For this C-tile, accumulate over k in blocks too.
			for kk := 0; kk < n; kk += block {
				kMax := min(kk+block, n)

				// Standard i-j-k tile loops.
				for i := ii; i < iMax; i++ {
					ai := i * n
					ci := i * n

					for k := kk; k < kMax; k++ {
						aik := a[ai+k] // A(i,k)
						if aik == 0 {
							continue
						}
						bk := k * n // row-major: B(k,j) lives in B[k*n + j]
						// Update C(i, j) for j in tile
						for j := jj; j < jMax; j++ {
							c[ci+j] += aik * b[bk+j]
						}
					}
				}
			}
		}
	}

	for w := 0; w < workers; w++ {
		go worker()
	}

	// Enqueue tiles. Each tile corresponds to a disjoint region of C.
	for ii := 0; ii < n; ii += block {
		for jj := 0; jj < n; jj += block {
			tasks <- tile{ii: ii, jj: jj}
		}
	}
	close(tasks)
	wg.Wait()

	return C, nil
}

//func min(a, b int) int {
//	if a < b {
//		return a
//	}
//	return b
//}
