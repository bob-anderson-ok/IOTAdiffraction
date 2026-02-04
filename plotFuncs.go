package main

import (
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	//"strconv"

	"gonum.org/v1/plot"

	// Liberation fonts register automatically on import
	_ "gonum.org/v1/plot/font/liberation"

	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

func makePlotImage(direction string, wPx, hPx float64, e OccultationEvent, edges []float64) (image.Image, error) {

	p := plot.New()

	p.Y.Min = -0.2
	p.Y.Max = 1.5

	X := 0
	Y := 1
	D := 2

	// Modify the font fields directly on existing styles
	p.Title.TextStyle.Font.Typeface = "Liberation"
	p.Title.TextStyle.Font.Variant = "Sans"
	p.Title.TextStyle.Font.Size = vg.Points(12)

	p.X.Label.TextStyle.Font.Typeface = "Liberation"
	p.X.Label.TextStyle.Font.Variant = "Sans"
	p.X.Label.TextStyle.Font.Size = vg.Points(12)

	p.Y.Label.TextStyle.Font.Typeface = "Liberation"
	p.Y.Label.TextStyle.Font.Variant = "Sans"
	p.Y.Label.TextStyle.Font.Size = vg.Points(12)

	p.X.Tick.Label.Font.Typeface = "Liberation"
	p.X.Tick.Label.Font.Variant = "Sans"
	p.X.Tick.Label.Font.Size = vg.Points(10)

	p.Y.Tick.Label.Font.Typeface = "Liberation"
	p.Y.Tick.Label.Font.Variant = "Sans"
	p.Y.Tick.Label.Font.Size = vg.Points(10)

	timePerPixel := e.FundamentalPlaneWidthKm / e.ShadowSpeedKmPerSec / float64(e.FundamentalPlaneWidthPoints)
	pointSpan := e.PathSamplePoints[len(e.PathSamplePoints)-1][D]
	distancePerPoint := e.FundamentalPlaneWidthKm / float64(e.FundamentalPlaneWidthPoints)
	timeSpan := timePerPixel * pointSpan
	fmt.Printf("Time span is %0.3f seconds\n", timeSpan)

	p.Title.Text = "Light curve along observation path"
	p.X.Label.Text = fmt.Sprintf("km (divide by the shadow speed of %0.3f km/second to get time)", e.ShadowSpeedKmPerSec)
	p.Y.Label.Text = "normalized intensity"
	p.X.Tick.Marker = StepTicks{Step: pointSpan * distancePerPoint / 20, Format: "%.2f"}

	p.Y.Tick.Marker = StepTicks{Step: 0.2, Format: "%.2f"}
	p.Add(plotter.NewGrid()) // grid + ticks

	//var reverse float64
	//var offset float64
	//
	//reverse = 1.0
	//offset = 0.0
	//
	//if direction == "right to left" {
	//	reverse = -1.0
	//	offset = e.PathSamplePoints[0][X] * timePerPixel
	//}
	//
	//if direction == "bottom to top" {
	//	reverse = -1.0
	//	offset = e.PathSamplePoints[0][Y] * timePerPixel
	//}

	// Data
	n := len(e.PathSamplePoints)
	pts := make(plotter.XYs, n)
	for i := 0; i < n; i++ {
		x := e.PathSamplePoints[i][X]
		y := e.PathSamplePoints[i][Y]
		intensity := interpolate(e.IntensityMatrix, x, y)
		pts[i].X = e.PathSamplePoints[i][D] * distancePerPoint
		pts[i].Y = intensity
	}

	line, err := plotter.NewLine(pts)
	if err != nil {
		return nil, err
	}
	line.Color = color.RGBA{R: 0, G: 0, B: 255, A: 255} // blue

	p.Add(line)

	if len(edges) > 0 {
		for _, edge := range edges {
			vpts := plotter.XYs{
				{X: edge * distancePerPoint, Y: -0.1},
				{X: edge * distancePerPoint, Y: 1.3},
			}

			vline, err := plotter.NewLine(vpts)
			if err != nil {
				panic(err)
			}

			p.Add(vline)
			//p.Legend.Add("signal", line)

			vline.Dashes = []vg.Length{
				vg.Points(6), // dash length
				vg.Points(4), // gap length
			}
			vline.Color = color.RGBA{R: 255, G: 0, B: 0, A: 255} // red
		}
	}

	hpts := plotter.XYs{
		{X: 0.0, Y: 0.0},
		{X: pointSpan * distancePerPoint, Y: 0.0},
	}

	hline, err := plotter.NewLine(hpts)
	if err != nil {
		panic(err)
	}

	p.Add(hline)

	hline.Dashes = []vg.Length{
		vg.Points(6), // dash length
		vg.Points(4), // gap length
	}
	hline.Color = color.RGBA{R: 0, G: 0, B: 0, A: 255} // black

	// Render into an in-memory image
	// Choose a "virtual" size in vg units and map to pixels via DPI.
	const dpi = 96
	width := vg.Length(wPx) * vg.Inch / dpi
	height := vg.Length(hPx) * vg.Inch / dpi

	c := vgimg.New(width, height)
	dc := draw.New(c)
	p.Draw(dc)

	return c.Image(), nil
}

type StepTicks struct {
	Step   float64
	Format string
}

func (t StepTicks) Ticks(min, max float64) []plot.Tick {
	var ticks []plot.Tick
	start := math.Ceil(min/t.Step) * t.Step
	for v := start; v <= max; v += t.Step {
		ticks = append(ticks, plot.Tick{
			Value: v,
			Label: fmt.Sprintf(t.Format, v),
		})
	}
	return ticks
}

func MakeCameraResponsePlot(data [][2]float64, filename string) {
	p := plot.New()

	// Modify the font fields directly on existing styles
	p.Title.TextStyle.Font.Typeface = "Liberation"
	p.Title.TextStyle.Font.Variant = "Sans"
	p.Title.TextStyle.Font.Size = vg.Points(12)

	p.X.Label.TextStyle.Font.Typeface = "Liberation"
	p.X.Label.TextStyle.Font.Variant = "Sans"
	p.X.Label.TextStyle.Font.Size = vg.Points(12)

	p.Y.Label.TextStyle.Font.Typeface = "Liberation"
	p.Y.Label.TextStyle.Font.Variant = "Sans"
	p.Y.Label.TextStyle.Font.Size = vg.Points(12)

	p.X.Tick.Label.Font.Typeface = "Liberation"
	p.X.Tick.Label.Font.Variant = "Sans"
	p.X.Tick.Label.Font.Size = vg.Points(10)

	p.Title.Text = "Camera response vs Wavelength from file: " + filename
	p.X.Label.Text = "Wavelength (nm)"
	p.Y.Label.Text = "Relative response"

	p.X.Tick.Marker = StepTicks{Step: 25.0, Format: "%.0f"}

	p.Y.Tick.Marker = StepTicks{Step: 0.1, Format: "%.2f"}
	p.Add(plotter.NewGrid()) // grid + ticks

	p.Y.Min = 0.0
	p.Y.Max = 1.1

	// Find the max weight - we will use that to calculate relative response
	var maxWeight = 0.0
	for _, pair := range data {
		if pair[1] > maxWeight {
			maxWeight = pair[1]
		}
	}
	// Data
	n := len(data)
	pts := make(plotter.XYs, n)
	for i := 0; i < n; i++ {
		x := data[i][0]
		y := data[i][1] / maxWeight
		pts[i].X = x
		pts[i].Y = y
	}

	linePoints, scatterPoints, err := plotter.NewLinePoints(pts)
	if err != nil {
		log.Fatal(err)
	}
	linePoints.Color = color.RGBA{R: 0, G: 0, B: 255, A: 255}
	linePoints.Width = vg.Points(1)

	scatterPoints.Shape = draw.CircleGlyph{}
	scatterPoints.Radius = vg.Points(2)
	scatterPoints.Color = color.RGBA{R: 120, G: 120, B: 120, A: 255}

	p.Add(linePoints, scatterPoints)

	hpts := plotter.XYs{
		{X: data[0][0], Y: 0.0},
		{X: data[n-1][0], Y: 0.0},
	}

	hline, err := plotter.NewLine(hpts)
	if err != nil {
		panic(err)
	}

	p.Add(hline)

	hline.Dashes = []vg.Length{
		vg.Points(6), // dash length
		vg.Points(4), // gap length
	}
	hline.Color = color.RGBA{R: 0, G: 0, B: 0, A: 255} // black

	if err := p.Save(8*vg.Inch, 4*vg.Inch, "camera_response.png"); err != nil {
		log.Fatal(err)
	}
	return
}
