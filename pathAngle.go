package main

import "math"

// PathAngleFromVelocity computes the observation path angle in degrees
// measured counter-clockwise from the positive Y-axis, given the shadow
// velocity components DxKmPerSec and DyKmPerSec. The result is in [0, 360).
func PathAngleFromVelocity(dxKmPerSec, dyKmPerSec float64) float64 {
	angle := math.Atan2(-dxKmPerSec, -dyKmPerSec) * 180.0 / math.Pi
	if angle < 0.0 {
		angle += 360.0
	}
	return angle
}
