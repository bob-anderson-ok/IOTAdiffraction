package main

import (
	"math"
	"math/cmplx"
)

func fresnelWeightsTopRow(NPts int, LKm, ZKm, WavelengthKm float64) []complex128 {

	// To calculate the fresnel weights matrix, we only need the top row
	// because all diagonals have a single value AND the matrix is symmetrical around the main diagonal,
	// so the transpose of this matrix equals the matrix.

	fresnelWeightsRow := make([]complex128, NPts)
	dx := LKm / float64(NPts)
	W := 1.0 / (2.0 * dx)
	delta := dx
	k := math.Pi * 2 / WavelengthKm
	x := Linspace(-LKm/2, LKm/2-dx, NPts)

	// The following terms are factors in the fresnel weight calculation that are not dependent on
	// either m or j in the for loop and so need only be calculated once.
	//
	// The sqrt(2.0/pi) term in t6 and the sqrt(pi/2.0) term in t4 are
	// needed because our implementation of the Fresnel integrals produces the 'normalized' form, but
	// Cabillos used the unnormalized form. Those terms perform the conversions.

	t1 := -math.Pi * math.Sqrt(2.0*ZKm/k) * W
	t2 := math.Sqrt(k / (2.0 * ZKm))
	t4 := (delta / math.Pi) * math.Sqrt(k/(2.0*ZKm)) * math.Sqrt(math.Pi/2.0)
	t5 := k / (2.0 * ZKm)
	t6 := math.Sqrt(2.0 / math.Pi)

	for m := range len(x) {
		slide := x[m] - x[0]
		u1x := t1 - t2*slide
		u2x := -t1 - t2*slide
		S1x, C1x := fresnelCephesScalar(u1x * t6)
		S2x, C2x := fresnelCephesScalar(u2x * t6)
		phiX := complex(t4, 0.0) * cmplx.Exp(complex(0.0, slide*slide*t5)) * complex(C2x-C1x, -(S2x-S1x))
		fresnelWeightsRow[m] = phiX
	}

	return fresnelWeightsRow
}

func fresnelWeights(NPts int, LKm, ZKm, WavelengthKm float64) [][]complex128 {

	// This routine is called when we want to see a full image of the diffraction pattern.
	// Usually, we only need to look at a single row, and there is a routine that does this
	// simpler task with a minimal use of memory: memory_frugal_single_row_sinc_solution().

	topRow := fresnelWeightsTopRow(NPts, LKm, ZKm, WavelengthKm)
	ans := make([][]complex128, NPts)
	for i := 0; i < NPts; i++ {
		ans[i] = make([]complex128, NPts)
	}

	// Build the full fresnel weights matrix from the top row
	for row := range NPts {
		for col := range NPts {
			ans[row][col] = topRow[AbsInt(col-row)] // element by element
		}
	}
	return ans
}

func AbsInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func FullObservationPlaneSincSolution(LKm, ZKm, WavelengthKm float64, sourcePlane [][]complex128) []complex128 {
	Npts := len(sourcePlane)
	wgts := fresnelWeights(Npts, LKm, ZKm, WavelengthKm)

	// k := math.Pi * 2.0 / WavelengthKm

	// The np.e**(1j * k * Z_km) term is not needed when the source and observation plane are far apart,
	// as they will be for asteroid occultations. Cabillos kept the term so that the solution was valid when
	// there was not much separation between the two planes. Removing the term makes applying Babinet to convert
	// an aperture to occulter simple using an incident wave as 1.0 + 0.0j
	// All that term does in astronomical usage is added a phase angle to the e-field that disappears
	// anyway when intensity is calculated by e_field * np.conj(e_field)

	M, N, K := Npts, Npts, Npts

	// BLAS uses column-major; leading dimensions:
	lda := M
	ldb := K
	ldc := M

	A, err := Flatten2D(wgts)
	if err != nil {
		panic(err)
	}

	B, err := Flatten2D(sourcePlane)
	if err != nil {
		panic(err)
	}

	C := make([]complex128, ldc*N)

	ans := make([]complex128, ldc*N)

	alpha := complex(1.0, 0.0)
	beta := complex(0.0, 0.0)

	// Compute wgts @ sourcePlane @ wgts
	Zgemm3m(Rowmajor, Notrans, Notrans, N, M, K, alpha, A, lda, B, ldb, beta, C, ldc)
	Zgemm3m(Rowmajor, Notrans, Notrans, N, M, K, alpha, C, lda, A, ldb, beta, ans, ldc)
	// Zgemm3m computes C <- alpha * A @ B + beta * C (which for us is C <- A @ B)

	return ans
}
