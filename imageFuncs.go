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
// This implements percentile stretch: map p Low to pHigh to 0..255 and clamp.
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
	startX, startY, endX, endY float64, drawStartDot bool) *image.RGBA {
	bounds := gray.Bounds()
	result := image.NewRGBA(bounds)
	draw.Draw(result, bounds, gray, bounds.Min, draw.Src)

	// Scale line width and dot radius to ~1% of the larger image dimension
	dim := bounds.Dx()
	if bounds.Dy() > dim {
		dim = bounds.Dy()
	}
	lineHalfWidth := float64(dim) / 400.0
	if lineHalfWidth < 1.5 {
		lineHalfWidth = 1.5
	}
	dotRadius := dim / 100
	if dotRadius < 5 {
		dotRadius = 5
	}

	// Draw the line in red
	drawLineOnImage(result, x1, y1, x2, y2, lineHalfWidth, color.RGBA{R: 255, A: 255})

	if drawStartDot {
		// Draw the start dot (red)
		drawDotOnImage(result, startX, startY, dotRadius, color.RGBA{R: 255, A: 255})
		// Draw the end dot (green)
		drawDotOnImage(result, endX, endY, dotRadius, color.RGBA{G: 255, A: 255})
	}

	return result
}

// drawLineOnImage draws an anti-aliased line with a given half-width.
// For each pixel in the bounding box of the line (plus margin), it computes
// the perpendicular distance to the line segment and blends accordingly.
func drawLineOnImage(img *image.RGBA, x1, y1, x2, y2, halfWidth float64, col color.Color) {

	r, g, b, a := col.RGBA()
	cr := uint8(r >> 8)
	cg := uint8(g >> 8)
	cb := uint8(b >> 8)
	ca := uint8(a >> 8)

	bounds := img.Bounds()
	margin := halfWidth + 1.5 // extra margin for antialiasing fringe
	minX := int(math.Floor(math.Min(x1, x2) - margin))
	maxX := int(math.Ceil(math.Max(x1, x2) + margin))
	minY := int(math.Floor(math.Min(y1, y2) - margin))
	maxY := int(math.Ceil(math.Max(y1, y2) + margin))
	if minX < bounds.Min.X {
		minX = bounds.Min.X
	}
	if minY < bounds.Min.Y {
		minY = bounds.Min.Y
	}
	if maxX > bounds.Max.X-1 {
		maxX = bounds.Max.X - 1
	}
	if maxY > bounds.Max.Y-1 {
		maxY = bounds.Max.Y - 1
	}

	// Line segment vector
	ldx := x2 - x1
	ldy := y2 - y1
	lenSq := ldx*ldx + ldy*ldy

	for py := minY; py <= maxY; py++ {
		for px := minX; px <= maxX; px++ {
			// Perpendicular distance from pixel center to line segment
			dist := distToSegment(float64(px), float64(py), x1, y1, ldx, ldy, lenSq)

			if dist > halfWidth+1.0 {
				continue
			}

			// Coverage: 1.0 inside the line, smooth falloff at edges
			coverage := math.Max(0, math.Min(1, halfWidth+0.5-dist))

			if coverage <= 0 {
				continue
			}

			// Alpha-blend the line color over the existing pixel
			bg := img.RGBAAt(px, py)
			alpha := float64(ca) / 255.0 * coverage
			inv := 1.0 - alpha
			nr := uint8(math.Min(255, float64(cr)*alpha+float64(bg.R)*inv))
			ng := uint8(math.Min(255, float64(cg)*alpha+float64(bg.G)*inv))
			nb := uint8(math.Min(255, float64(cb)*alpha+float64(bg.B)*inv))
			na := uint8(math.Min(255, float64(ca)*alpha+float64(bg.A)*inv+alpha*255))
			img.Pix[img.PixOffset(px, py)] = nr
			img.Pix[img.PixOffset(px, py)+1] = ng
			img.Pix[img.PixOffset(px, py)+2] = nb
			img.Pix[img.PixOffset(px, py)+3] = na
		}
	}
}

// distToSegment returns the perpendicular distance from point (px, py) to the
// line segment defined by start (sx, sy) and direction (dx, dy) with squared
// length lenSq.
func distToSegment(px, py, sx, sy, dx, dy, lenSq float64) float64 {
	if lenSq == 0 {
		return math.Hypot(px-sx, py-sy)
	}
	// Project point onto the line, clamping t to [0,1] for segment
	t := ((px-sx)*dx + (py-sy)*dy) / lenSq
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	closestX := sx + t*dx
	closestY := sy + t*dy
	return math.Hypot(px-closestX, py-closestY)
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

// drawPsfOnImage overlays a PSF matrix onto an RGBA image centered at (cx, cy)
// using the given color. Brightness of each PSF pixel is normalized by maxVal
// and used as the alpha blend factor.
func drawPsfOnImage(img *image.RGBA, cx, cy float64, psf [][]float64, col color.RGBA) {
	psfRows := len(psf)
	if psfRows == 0 {
		return
	}
	psfCols := len(psf[0])

	// Find the max value in the PSF for normalization
	maxVal := 0.0
	for _, row := range psf {
		for _, v := range row {
			if v > maxVal {
				maxVal = v
			}
		}
	}
	if maxVal <= 0 {
		return
	}

	bounds := img.Bounds()
	centerRow := float64(psfRows) / 2.0
	centerCol := float64(psfCols) / 2.0

	// Offset the draw position so the entire PSF stays within the image
	if cx-centerCol < float64(bounds.Min.X) {
		cx = float64(bounds.Min.X) + centerCol
	}
	if cy-centerRow < float64(bounds.Min.Y) {
		cy = float64(bounds.Min.Y) + centerRow
	}
	if cx+centerCol > float64(bounds.Max.X) {
		cx = float64(bounds.Max.X) - centerCol
	}
	if cy+centerRow > float64(bounds.Max.Y) {
		cy = float64(bounds.Max.Y) - centerRow
	}

	for row := 0; row < psfRows; row++ {
		for c := 0; c < psfCols; c++ {
			if psf[row][c] <= 0 {
				continue
			}
			px := int(cx + float64(c) - centerCol)
			py := int(cy + float64(row) - centerRow)
			if px < bounds.Min.X || px >= bounds.Max.X || py < bounds.Min.Y || py >= bounds.Max.Y {
				continue
			}
			alpha := psf[row][c] / maxVal
			bg := img.RGBAAt(px, py)
			blended := color.RGBA{
				R: uint8(float64(bg.R)*(1-alpha) + float64(col.R)*alpha),
				G: uint8(float64(bg.G)*(1-alpha) + float64(col.G)*alpha),
				B: uint8(float64(bg.B)*(1-alpha) + float64(col.B)*alpha),
				A: 255,
			}
			img.SetRGBA(px, py, blended)
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
