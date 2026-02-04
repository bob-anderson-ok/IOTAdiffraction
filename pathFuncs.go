package main

import (
	"errors"
	"fmt"
	"math"
)

func processPathDirection(Npts int, p1 AnnotatedPoint, p2 AnnotatedPoint,
	event *OccultationEvent) (AnnotatedPoint, AnnotatedPoint, string, error) {
	w := float64(Npts - 1)
	direction := "unknown"

	theta := event.PathAngleDegrees * math.Pi / 180.0
	// positive d moves the path to the right from the perspective of someone riding with the star
	// in the movement direction
	d := (event.PathOffsetFromCenterKm / event.FundamentalPlaneWidthKm) * float64(event.FundamentalPlaneWidthPoints)
	dx := 0.0
	dy := 0.0
	p1, p2, dx, dy, err := PathSquareIntersections(w, theta, d)
	fmt.Printf("\nDirection vector of path in image coordinates: dx=%.4f dy=%.4f\n\n", dx, dy)

	if err != nil {
		fmt.Println("Error:", err)
	} else {
		// Move the origin back to the upper left corner of the image
		delta := float64(Npts) / 2.0
		p1.X += delta
		p1.Y += delta
		p2.X += delta
		p2.Y += delta
		fmt.Printf("Intersection 1: (%.4f, %.4f)  %s\n", p1.X, p1.Y, p1.Position)
		fmt.Printf("Intersection 2: (%.4f, %.4f)  %s\n", p2.X, p2.Y, p2.Position)

		// Time to figure out the direction and fill start and end coordinates
		useTopBottomLogic := (p1.Position == "top" || p1.Position == "bottom") &&
			(p2.Position == "top" || p2.Position == "bottom")
		if useTopBottomLogic {
			if dy < 0 {
				direction = "top to bottom"
				if p1.Position == "top" {
					setPathStartEnd(event, p1, p2)
				} else {
					setPathStartEnd(event, p2, p1)
				}
			} else {
				direction = "bottom to top"
				if p1.Position == "bottom" {
					setPathStartEnd(event, p1, p2)
				} else {
					setPathStartEnd(event, p2, p1)
				}
			}
		} else {
			if dx < 0 {
				direction = "left to right"
				if p1.Position == "left" {
					setPathStartEnd(event, p1, p2)
				} else {
					setPathStartEnd(event, p2, p1)
				}
			} else {
				direction = "right to left"
				if p1.Position == "right" {
					setPathStartEnd(event, p1, p2)
				} else {
					setPathStartEnd(event, p2, p1)
				}
			}
		}
		fmt.Println("\nPath start:", event.PathStart)
		fmt.Println("Path end:", event.PathEnd)
	}
	return p1, p2, direction, err
}

func FindEdgesInGeometricShadow(e OccultationEvent) []float64 {
	var ans []float64
	var colorAtNextEdge = 1.0

	for i := range len(e.PathSamplePoints) {
		pixelValue := interpolate(e.GeometricMatrix, e.PathSamplePoints[i][0], e.PathSamplePoints[i][1])
		if pixelValue > 0.0 {
			pixelValue = 1.0
		}
		if pixelValue == colorAtNextEdge {
			ans = append(ans, e.PathSamplePoints[i][2]) // append the distanceFromStart value
			colorAtNextEdge = 1.0 - colorAtNextEdge     // Toggle the color we treat as an edge
		}
	}
	return ans
}

func setPathStartEnd(event *OccultationEvent, pStart AnnotatedPoint, pEnd AnnotatedPoint) {
	event.PathStart[0] = pStart.X
	event.PathStart[1] = pStart.Y
	event.PathEnd[0] = pEnd.X
	event.PathEnd[1] = pEnd.Y
}

type AnnotatedPoint struct {
	X, Y     float64
	Position string
}

var ErrNoIntersection = errors.New("line does not intersect square")

// PathSquareIntersections finds where a line intersects a square centered at origin.
// w: square width
// theta: angle of line measured CCW from y-axis (radians)
// d: perpendicular distance from point (px, py) to the line
// Returns the two intersection points, dx and dy, and error
func PathSquareIntersections(w, theta, d float64) (AnnotatedPoint, AnnotatedPoint, float64, float64, error) {
	halfW := w / 2.0

	// Direction vector of the line (perpendicular to the normal)
	// If theta is CCW from y-axis, the line direction is (sin(theta), cos(theta))
	dx := math.Sin(theta)
	dy := math.Cos(theta)

	// Normal vector pointing in the direction of offset (perpendicular to line, rotated 90° CW)
	nx := dy  // cos(theta)
	ny := -dx // -sin(theta)

	// A point on the line: offset from (px, py) by distance d along the normal
	x0 := d * nx
	y0 := d * ny

	// Line parametric form: x = x0 + t*dx, y = y0 + t*dy
	// Find intersections with the four sides of the square

	var intersections []AnnotatedPoint

	// Right edge: x = halfW
	if math.Abs(dx) > 1e-12 {
		t := (halfW - x0) / dx
		y := y0 + t*dy
		if y >= -halfW && y <= halfW {
			intersections = append(intersections, AnnotatedPoint{halfW, y, "right"})
		}
	}

	// Left edge: x = -halfW
	if math.Abs(dx) > 1e-12 {
		t := (-halfW - x0) / dx
		y := y0 + t*dy
		if y >= -halfW && y <= halfW {
			intersections = append(intersections, AnnotatedPoint{-halfW, y, "left"})
		}
	}

	// Bottom edge: y = halfW
	if math.Abs(dy) > 1e-12 {
		t := (halfW - y0) / dy
		x := x0 + t*dx
		if x >= -halfW && x <= halfW {
			intersections = append(intersections, AnnotatedPoint{x, halfW, "bottom"})
		}
	}

	// Top edge: y = halfW
	if math.Abs(dy) > 1e-12 {
		t := (-halfW - y0) / dy
		x := x0 + t*dx
		if x >= -halfW && x <= halfW {
			intersections = append(intersections, AnnotatedPoint{x, -halfW, "top"})
		}
	}

	// Remove duplicate corner intersections
	intersections = removeDuplicates(intersections, 1e-9)

	if len(intersections) < 2 {
		return AnnotatedPoint{}, AnnotatedPoint{}, dx, dy, ErrNoIntersection
	}
	return intersections[0], intersections[1], dx, dy, nil
}

func removeDuplicates(pts []AnnotatedPoint, tol float64) []AnnotatedPoint {
	var result []AnnotatedPoint
	for _, p := range pts {
		duplicate := false
		for _, r := range result {
			if math.Abs(p.X-r.X) < tol && math.Abs(p.Y-r.Y) < tol {
				duplicate = true
				break
			}
		}
		if !duplicate {
			result = append(result, p)
		}
	}
	return result
}
