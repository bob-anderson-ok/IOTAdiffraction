package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"math/rand"
	"os"
	"sort"
)

func interpolate(matrix [][]float64, x, y float64) float64 {
	n := len(matrix)
	if n == 0 {
		return 0
	}

	// Clamp to valid range (that is, at the edges of matrix
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x >= float64(n-1) {
		x = float64(n-1) - 1e-9
	}
	if y >= float64(n-1) {
		y = float64(n-1) - 1e-9
	}

	// Integer indices
	x0 := int(x)
	y0 := int(y)
	x1 := x0 + 1
	y1 := y0 + 1

	// Fractional parts
	xFrac := x - float64(x0)
	yFrac := y - float64(y0)

	// Four surrounding values
	v00 := matrix[y0][x0]
	v01 := matrix[y0][x1]
	v10 := matrix[y1][x0]
	v11 := matrix[y1][x1]

	// Bilinear interpolation
	v0 := v00*(1-xFrac) + v01*xFrac
	v1 := v10*(1-xFrac) + v11*xFrac

	return v0*(1-yFrac) + v1*yFrac
}

func addScaledComplexInPlace(a []complex128, b []complex128, scaleB float64) {
	if len(a) != len(b) {
		panic("vector lengths don't match")
	}

	for i := range a {
		a[i] = a[i] + complex(scaleB, 0)*b[i]
	}
}

func scaleComplex(v []complex128, scale float64) {
	s := complex(scale, 0)
	for i := range v {
		v[i] *= s
	}
}

// -------------------- I/O --------------------

//func SavePNG(path string, img image.Image) error {
//	f, err := os.Create(path)
//	if err != nil {
//		return err
//	}
//	defer f.Close()
//	return png.Encode(f, img)
//}

// MatrixToGray16Data -------------------- Data PNG (Gray16, fixed physical scaling) --------------------
// Mapping: Y16 = round(v * scale), clamped to [0, 65535]
func MatrixToGray16Data(m [][]float64, scale float64) (*image.Gray16, error) {
	if len(m) == 0 || len(m[0]) == 0 {
		return nil, errors.New("empty matrix")
	}
	if scale <= 0 {
		return nil, errors.New("scale must be > 0")
	}
	h := len(m)
	w := len(m[0])
	for y := 1; y < h; y++ {
		if len(m[y]) != w {
			return nil, errors.New("ragged matrix")
		}
	}

	img := image.NewGray16(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		row := y * img.Stride
		for x := 0; x < w; x++ {
			v := m[y][x]
			if math.IsNaN(v) || math.IsInf(v, 0) {
				// write 0
				i := row + 2*x
				img.Pix[i], img.Pix[i+1] = 0, 0
				continue
			}

			u := math.Round(v * scale)
			if u < 0 {
				u = 0
			} else if u > 65535 {
				u = 65535
			}
			y16 := uint16(u)

			// Gray16 Pix is big-endian per pixel: high then low
			i := row + 2*x
			img.Pix[i] = uint8(y16 >> 8)
			img.Pix[i+1] = uint8(y16)
		}
	}
	return img, nil
}

// MatrixToGrayViewPercentile -------------------- View PNG (Gray8, auto-stretch) --------------------
// Two common auto-stretches:
//
//	A) Min/Max stretch (simple)
//	B) Percentile stretch (robust to outliers) <-- recommended
//
// This implements percentile stretch: map pLow to pHigh to 0..255 and clamp.
func MatrixToGrayViewPercentile(m [][]float64, pLow, pHigh float64) (*image.Gray, error) {
	if len(m) == 0 || len(m[0]) == 0 {
		return nil, errors.New("empty matrix")
	}
	h := len(m)
	w := len(m[0])
	for y := 1; y < h; y++ {
		if len(m[y]) != w {
			return nil, errors.New("ragged matrix")
		}
	}
	if !(0 <= pLow && pLow < pHigh && pHigh <= 100) {
		return nil, errors.New("percentiles must satisfy 0 <= p Low < pHigh <= 100")
	}

	// Collect finite values for percentile computation
	vals := make([]float64, 0, h*w)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := m[y][x]
			if !math.IsNaN(v) && !math.IsInf(v, 0) {
				vals = append(vals, v)
			}
		}
	}
	if len(vals) == 0 {
		return nil, errors.New("matrix has no finite values")
	}

	sort.Float64s(vals)

	// Helper to get percentile value
	percentile := func(p float64) float64 {
		if p <= 0 {
			return vals[0]
		}
		if p >= 100 {
			return vals[len(vals)-1]
		}
		pos := (p / 100.0) * float64(len(vals)-1)
		i := int(math.Floor(pos))
		f := pos - float64(i)
		if i >= len(vals)-1 {
			return vals[len(vals)-1]
		}
		return vals[i]*(1-f) + vals[i+1]*f
	}

	lo := percentile(pLow)
	hi := percentile(pHigh)
	if hi == lo {
		hi = lo + 1 // avoid divide-by-zero; image becomes mostly constant
	}

	img := image.NewGray(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		row := y * img.Stride
		for x := 0; x < w; x++ {
			v := m[y][x]
			if math.IsNaN(v) || math.IsInf(v, 0) {
				img.Pix[row+x] = 0
				continue
			}
			t := (v - lo) / (hi - lo) // normalize
			if t < 0 {
				t = 0
			} else if t > 1 {
				t = 1
			}
			img.Pix[row+x] = uint8(math.Round(t * 255.0))
		}
	}
	return img, nil
}

func SaveGrayPNG(filename string, img *image.Gray) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	//defer f.Close()
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()
	return png.Encode(f, img)
}

func SaveGray16PNG(filename string, img *image.Gray16) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	//defer f.Close()
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	return png.Encode(f, img)
}

func fillComplex(rng *rand.Rand, x []complex128) {
	for i := range x {
		// keep magnitudes moderate to avoid overflow in large GEMMs
		re := (rng.Float64() - 0.5) * 2.0
		im := (rng.Float64() - 0.5) * 2.0
		x[i] = complex(re, im)
	}
}

func ConvertSourcePlaneImageToComplex(img *image.Gray) [][]complex128 {
	m := make([][]complex128, img.Bounds().Dy())
	for y := 0; y < img.Bounds().Dy(); y++ {
		m[y] = make([]complex128, img.Bounds().Dx())
		for x := 0; x < img.Bounds().Dx(); x++ {
			if img.GrayAt(x, y).Y == 0 {
				m[y][x] = complex(1.0, 0.0) // We create an aperture from the black on white image
			} else {
				m[y][x] = complex(0.0, 0.0)
			}
		}
	}
	return m
}

func ConvertSourcePlaneImageToMatrix(img *image.Gray) [][]float64 {
	m := make([][]float64, img.Bounds().Dy())
	for y := 0; y < img.Bounds().Dy(); y++ {
		m[y] = make([]float64, img.Bounds().Dx())
		for x := 0; x < img.Bounds().Dx(); x++ {
			if img.GrayAt(x, y).Y == 0 {
				m[y][x] = 1.0 // We create an aperture from the black on white image
			} else {
				m[y][x] = 0.0
			}
		}
	}
	return m
}

func FillFplane(img *image.Gray, occulterWanted bool) {
	var fill uint8

	if occulterWanted {
		fill = 255
	} else {
		fill = 0
	}
	for y := 0; y < img.Rect.Dy(); y++ {
		row := y * img.Stride
		for x := 0; x < img.Rect.Dx(); x++ {
			img.Pix[row+x] = fill
		}
	}
}
func Flatten2D(m [][]complex128) ([]complex128, error) {
	// Row major flattening
	rows := len(m)
	if rows == 0 {
		return nil, nil
	}
	cols := len(m[0])

	// Ensure rectangular
	for i := 1; i < rows; i++ {
		if len(m[i]) != cols {
			return nil, fmt.Errorf("ragged matrix")
		}
	}

	out := make([]complex128, rows*cols)
	k := 0
	for i := 0; i < rows; i++ {
		copy(out[k:k+cols], m[i])
		k += cols
	}
	return out, nil
}

//func ReshapeComplex1DTo2D(v []complex128, rows, cols int) ([][]complex128, error) {
//	if len(v) != rows*cols {
//		return nil, fmt.Errorf("size mismatch: have %d, want %d", len(v), rows*cols)
//	}
//
//	m := make([][]complex128, rows)
//	k := 0
//	for i := 0; i < rows; i++ {
//		m[i] = make([]complex128, cols)
//		copy(m[i], v[k:k+cols])
//		k += cols
//	}
//	return m, nil
//}

// DrawPathOnImage draws the observation path on a grayscale image and returns a new RGBA image.
// The path line is drawn in red from (x1,y1) to (x2,y2).
// A red dot is drawn at the start point and a green dot at the end point.
func DrawPathOnImage(gray *image.Gray, x1, y1, x2, y2 float64,
	startX, startY, endX, endY float64) *image.RGBA {
	bounds := gray.Bounds()
	result := image.NewRGBA(bounds)
	draw.Draw(result, bounds, gray, bounds.Min, draw.Src)

	// Draw the line in red
	drawLineOnImage(result, x1, y1, x2, y2, color.RGBA{R: 255, A: 255})

	// Draw the start dot (red)
	drawDotOnImage(result, startX, startY, 5, color.RGBA{R: 255, A: 255})

	// Draw the end dot (green)
	drawDotOnImage(result, endX, endY, 5, color.RGBA{G: 255, A: 255})

	return result
}

// drawLineOnImage draws a line using Bresenham's algorithm with 3-pixel width.
func drawLineOnImage(img *image.RGBA, x1, y1, x2, y2 float64, col color.Color) {
	dx := math.Abs(x2 - x1)
	dy := math.Abs(y2 - y1)
	sx := -1.0
	if x1 < x2 {
		sx = 1.0
	}
	sy := -1.0
	if y1 < y2 {
		sy = 1.0
	}
	err := dx - dy

	for {
		for oy := -1; oy <= 1; oy++ {
			for ox := -1; ox <= 1; ox++ {
				px := int(x1) + ox
				py := int(y1) + oy
				if px >= 0 && px < img.Bounds().Dx() && py >= 0 && py < img.Bounds().Dy() {
					img.Set(px, py, col)
				}
			}
		}

		if math.Abs(x1-x2) < 1 && math.Abs(y1-y2) < 1 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}
}

// drawDotOnImage draws a filled circle on the image.
func drawDotOnImage(img *image.RGBA, cx, cy float64, radius int, col color.Color) {
	for y := -radius; y <= radius; y++ {
		for x := -radius; x <= radius; x++ {
			if x*x+y*y <= radius*radius {
				px := int(cx) + x
				py := int(cy) + y
				if px >= 0 && px < img.Bounds().Dx() && py >= 0 && py < img.Bounds().Dy() {
					img.Set(px, py, col)
				}
			}
		}
	}
}

// SaveImagePNG saves any image.Image to a PNG file.
func SaveImagePNG(filename string, img image.Image) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()
	return png.Encode(f, img)
}

func Reshape1DTo2D(v []float64, rows, cols int) ([][]float64, error) {
	if len(v) != rows*cols {
		return nil, fmt.Errorf("size mismatch: have %d, want %d", len(v), rows*cols)
	}

	m := make([][]float64, rows)
	k := 0
	for i := 0; i < rows; i++ {
		m[i] = make([]float64, cols)
		copy(m[i], v[k:k+cols])
		k += cols
	}
	return m, nil
}

//func View1DAs2D(v []complex128, rows, cols int) ([][]complex128, error) {
//	if len(v) != rows*cols {
//		return nil, fmt.Errorf("size mismatch")
//	}
//	m := make([][]complex128, rows)
//	for i := 0; i < rows; i++ {
//		m[i] = v[i*cols : (i+1)*cols]
//	}
//	return m, nil
//}
