package main

import (
	"fmt"
	"image/color"
	"math"
)

func insideGeneralizedEllipse(x, y, x0, y0, xDiam, yDiam, thetaDegrees float64) bool {
	//#Returns boolean: True if x,y is inside or on the ellipse boundary - False if x,y is outside the ellipse
	// The fundamental plane coordinate system is y (row) pointing up and x (column) pointing left.
	// x0,y0 are the coordinates of the center of the ellipse.
	// theta_degrees is the counter-clockwise rotation (in degrees) around x0,y0 with North at zero degrees
	xSemi := xDiam / 2.0
	ySemi := yDiam / 2.0

	// Change center coordinates to deal with coordinates specified in the fundamental plane scheme
	xc := -y0
	yc := -x0
	thetaRadians := (thetaDegrees + 90.0) * (math.Pi / 180.0) // Add 90.0 so angles are ccw from North
	t1 := ((x-xc)*math.Cos(thetaRadians) + (y+yc)*math.Sin(thetaRadians)) / xSemi
	t2 := ((-x+xc)*math.Sin(thetaRadians) + (y+yc)*math.Cos(thetaRadians)) / ySemi
	return t1*t1+t2*t2 <= 1.0
}

func ColorModelString(m color.Model) string {
	switch m {
	case color.RGBAModel:
		return "RGBA"
	case color.NRGBAModel:
		return "NRGBA"
	case color.GrayModel:
		return "Gray"
	case color.Gray16Model:
		return "Gray16"
	case color.CMYKModel:
		return "CMYK"
	case color.AlphaModel:
		return "Alpha"
	case color.Alpha16Model:
		return "Alpha16"
	default:
		return fmt.Sprintf("Unknown (%T)", m)
	}
}

// Linspace This is provided to match numpy's linspace()
func Linspace(start, end float64, n int) []float64 {
	if n <= 1 {
		return []float64{start}
	}

	step := (end - start) / float64(n-1)

	x := make([]float64, n)
	for i := 0; i < n; i++ {
		x[i] = start + float64(i)*step
	}
	return x
}

func AddEllipses(event OccultationEvent, occulter bool) {
	if !(event.MainBodyGiven || event.SatelliteGiven) {
		return
	}

	var objectFill uint8
	if occulter {
		objectFill = 0
	} else {
		objectFill = 255
	}

	// In the fundamental plane, x is most positive at the left.
	xVals := Linspace(
		event.FundamentalPlaneWidthKm/2,
		-event.FundamentalPlaneWidthKm/2,
		event.FundamentalPlaneWidthPoints,
	)

	// In the fundamental plane, y is most positive at the top.
	yVals := Linspace(
		event.FundamentalPlaneWidthKm/2,
		-event.FundamentalPlaneWidthKm/2,
		event.FundamentalPlaneWidthPoints,
	)

	var x0, y0, xDiam, yDiam, rotation float64

	if event.MainBodyGiven {
		x0 = -event.MainBodyXCenterKm
		y0 = -event.MainBodyYCenterKm
		xDiam = event.MainbodyMinorAxisKm
		yDiam = event.MainbodyMajorAxisKm
		rotation = event.MainbodyMajorAxisPaDegrees

		for row := 0; row < event.FundamentalPlaneWidthPoints; row++ {
			for col := 0; col < event.FundamentalPlaneWidthPoints; col++ {
				if insideGeneralizedEllipse(xVals[col], yVals[row], x0, y0, xDiam, yDiam, rotation) {
					event.FplaneImage.Set(row, col, color.Gray{Y: objectFill})
				}
			}
		}
	}

	if event.SatelliteGiven {
		x0 = -event.SatelliteXCenterKm
		y0 = -event.SatelliteYCenterKm
		xDiam = event.SatelliteMinorAxisKm
		yDiam = event.SatelliteMajorAxisKm
		rotation = event.SatelliteMajorAxisPaDegrees

		for row := 0; row < event.FundamentalPlaneWidthPoints; row++ {
			for col := 0; col < event.FundamentalPlaneWidthPoints; col++ {
				if insideGeneralizedEllipse(xVals[col], yVals[row], x0, y0, xDiam, yDiam, rotation) {
					event.FplaneImage.Set(row, col, color.Gray{Y: objectFill})
				}
			}
		}
	}
}
