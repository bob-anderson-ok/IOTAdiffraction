package main

import (
	"errors"
	"math"

	"gonum.org/v1/gonum/dsp/fourier"
)

type ConvMode int

const (
	ConvSame ConvMode = iota
	ConvFull
	ConvValid
)

type PaddingMode int

const (
	PadZeros PaddingMode = iota
	PadReflect
	PadReplicate
	PadCircular
)

func StarBrightness(r, starDiamKm, limbDarkeningCoeff float64) float64 {
	starRadius := starDiamKm / 2.0
	// x is the distance from the star center expressed as a fraction of the star radius
	x := r / starRadius
	if x >= 1.0 {
		return 0.0
	}

	return 1.0 - limbDarkeningCoeff*(1.0-math.Sqrt(1.0-x*x))
}

func BuildStarPsf(starDiamKm, resolutionPointsPerKm, limbDarkeningCoeff float64) ([][]float64, float64) {
	// First, we compute the dimensions of the enclosing square.
	psfWidthPixels := int(math.Ceil(starDiamKm / resolutionPointsPerKm))
	//fmt.Printf("\nStarWidthPixels = %d\n", psfWidthPixels)
	// Make psfWidth even.
	if psfWidthPixels%2 != 0 {
		psfWidthPixels++
	}
	// add a border
	psfWidthPixels += 4
	starMatrix := make([][]float64, psfWidthPixels)
	center := psfWidthPixels / 2
	sumOfWeights := 0.0
	for row := 0; row < psfWidthPixels; row++ {
		for col := 0; col < psfWidthPixels; col++ {
			r := math.Sqrt(float64((row-center)*(row-center)+(col-center)*(col-center))) * resolutionPointsPerKm
			brightness := StarBrightness(r, starDiamKm, limbDarkeningCoeff)
			sumOfWeights += brightness
			starMatrix[row] = append(starMatrix[row], brightness)
		}
	}
	return starMatrix, sumOfWeights
}

// ConvolvePSFFFT convolves image with a centered PSF using 2D FFT.
//
// image: HxW
// psf:   PhxPw (assumed centered; peak near middle)
// mode:  Same, Full, Valid
// pad:   Zeros, Reflect, Replicate, Circular
//
// Returns a real-valued output (same units as input).
func ConvolvePSFFFT(image, psf [][]float64, starSum float64, mode ConvMode, pad PaddingMode, centeredPsf bool) ([][]float64, error) {
	H, W, err := rectSize(image)
	if err != nil {
		return nil, err
	}
	Ph, Pw, err := rectSize(psf)
	if err != nil {
		return nil, err
	}
	if H == 0 || W == 0 || Ph == 0 || Pw == 0 {
		return nil, errors.New("empty image or psf")
	}

	// Output dimensions.
	var outH, outW int
	switch mode {
	case ConvSame:
		outH, outW = H, W
	case ConvFull:
		outH, outW = H+Ph-1, W+Pw-1
	case ConvValid:
		outH, outW = H-Ph+1, W-Pw+1
		if outH <= 0 || outW <= 0 {
			return nil, errors.New("valid convolution requested but psf larger than image")
		}
	default:
		return nil, errors.New("unknown ConvMode")
	}

	// FFT grid for linear convolution: at least full size.
	// You may choose the nextPow2 for speed; Gonum works for any n, but pow2 is often faster.
	FH := nextPow2(H + Ph - 1)
	FW := nextPow2(W + Pw - 1)

	//FH = H + Ph - 1
	//FW = W + Pw - 1

	if FH%2 != 0 {
		FH++
	}
	if FW%2 != 0 {
		FW++
	}

	//fmt.Printf("FH = %d, H = %d, Ph = %d\n", FH, H, Ph)
	//fmt.Printf("FW = %d, W = %d, Pw = %d\n", FW, W, Pw)

	// Build padded complex grids.
	A := makeComplex2D(FH, FW)
	B := makeComplex2D(FH, FW)

	// Put the image into A with a chosen padding policy.
	// For linear convolution, we conceptually embed the original image in the top-left
	// of the FFT grid. Padding is only relevant if you want boundary-handling other than zeros.
	for y := 0; y < FH; y++ {
		for x := 0; x < FW; x++ {
			A[y][x] = complex(sample2D(image, y, x, pad), 0)
		}
	}

	if centeredPsf {
		// Put PSF into B, but shift it so its center is at (0,0) (ifftshift).
		// This is essential if your PSF is stored "centered", which is typical.
		psfShift := ifftshift2D(psf)

		for y := 0; y < Ph; y++ {
			for x := 0; x < Pw; x++ {
				B[y][x] = complex(psfShift[y][x], 0)
			}
		}
	} else {
		for y := 0; y < Ph; y++ {
			for x := 0; x < Pw; x++ {
				B[y][x] = complex(psf[y][x], 0)
			}
		}
	}

	// 2D FFT: rows then columns using Gonum CmplxFFT. :contentReference[oaicite:2]{index=2}
	fft2InPlace(A, true)
	fft2InPlace(B, true)

	// Multiply spectra.
	for y := 0; y < FH; y++ {
		for x := 0; x < FW; x++ {
			A[y][x] *= B[y][x]
		}
	}

	// Inverse 2D FFT.
	fft2InPlace(A, false)

	// Gonum transforms are unnormalized: forward then inverse multiplies by N. contentReference[oaicite:3]{index=3}
	// For 2D, divide by FH*FW.
	scale := float64(FH * FW)

	// Crop according to ConvMode.
	full := make([][]float64, H+Ph-1)
	for y := range full {
		full[y] = make([]float64, W+Pw-1)
		for x := range full[y] {
			full[y][x] = real(A[y][x]) / scale / starSum
		}
	}

	switch mode {
	case ConvFull:
		return full, nil

	case ConvSame:
		// Centered crop of a full result to HxW: offset = floor(Ph/2), floor(Pw/2)
		offY := Ph / 2
		offX := Pw / 2
		out := make([][]float64, H)
		for y := 0; y < H; y++ {
			out[y] = make([]float64, W)
			copy(out[y], full[y+offY][offX:offX+W])
		}
		return out, nil

	case ConvValid:
		// Valid crop of a full result: start at (Ph-1, Pw-1)
		startY := Ph - 1
		startX := Pw - 1
		out := make([][]float64, outH)
		for y := 0; y < outH; y++ {
			out[y] = make([]float64, outW)
			copy(out[y], full[y+startY][startX:startX+outW])
		}
		return out, nil
	}

	return nil, errors.New("unreachable")
}

// -------------------- FFT helpers --------------------

func fft2InPlace(a [][]complex128, forward bool) {
	h := len(a)
	w := len(a[0])

	rowFFT := fourier.NewCmplxFFT(w)
	colFFT := fourier.NewCmplxFFT(h)

	// rows
	tmp := make([]complex128, w)
	for y := 0; y < h; y++ {
		copy(tmp, a[y])
		if forward {
			rowFFT.Coefficients(tmp, tmp)
		} else {
			rowFFT.Sequence(tmp, tmp)
		}
		copy(a[y], tmp)
	}

	// cols
	col := make([]complex128, h)
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			col[y] = a[y][x]
		}
		if forward {
			colFFT.Coefficients(col, col)
		} else {
			colFFT.Sequence(col, col)
		}
		for y := 0; y < h; y++ {
			a[y][x] = col[y]
		}
	}
}

// -------------------- Padding + shifting --------------------

func sample2D(img [][]float64, y, x int, mode PaddingMode) float64 {
	H := len(img)
	W := len(img[0])

	if 0 <= y && y < H && 0 <= x && x < W {
		return img[y][x]
	}

	switch mode {
	case PadZeros:
		return 0

	case PadReplicate:
		yy := clamp(y, 0, H-1)
		xx := clamp(x, 0, W-1)
		return img[yy][xx]

	case PadReflect:
		yy := reflectIndex(y, H)
		xx := reflectIndex(x, W)
		return img[yy][xx]

	case PadCircular:
		yy := mod(y, H)
		xx := mod(x, W)
		return img[yy][xx]
	}

	return 0
}

// ifftshift: moves the center of a centered PSF to (0,0).
func ifftshift2D(x [][]float64) [][]float64 {
	h := len(x)
	w := len(x[0])
	out := make([][]float64, h)
	for i := range out {
		out[i] = make([]float64, w)
	}
	shY := h / 2
	shX := w / 2
	for y := 0; y < h; y++ {
		yy := (y + shY) % h
		for x0 := 0; x0 < w; x0++ {
			xx := (x0 + shX) % w
			out[y][x0] = x[yy][xx]
		}
	}
	return out
}

// -------------------- utility --------------------

func rectSize(m [][]float64) (h, w int, err error) {
	h = len(m)
	if h == 0 {
		return 0, 0, nil
	}
	w = len(m[0])
	for i := 1; i < h; i++ {
		if len(m[i]) != w {
			return 0, 0, errors.New("ragged matrix")
		}
	}
	return h, w, nil
}

func makeComplex2D(h, w int) [][]complex128 {
	m := make([][]complex128, h)
	for i := range m {
		m[i] = make([]complex128, w)
	}
	return m
}

func nextPow2(n int) int {
	if n <= 1 {
		return 1
	}
	p := 1
	for p < n {
		p <<= 1
	}
	return p
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func mod(i, n int) int {
	r := i % n
	if r < 0 {
		r += n
	}
	return r
}

// reflectIndex implements "reflect" padding without repeating edge pixels.
// Example for n=5 indices: ... 2 1 0 1 2 3 4 3 2 1 0 1 ...
func reflectIndex(i, n int) int {
	if n <= 1 {
		return 0
	}
	period := 2*n - 2
	i = mod(i, period)
	if i >= n {
		i = period - i
	}
	return i
}

// Optional: avoid tiny negative zeros if you care.
func cleanZero(x float64) float64 {
	if math.Abs(x) < 1e-15 {
		return 0
	}
	return x
}
