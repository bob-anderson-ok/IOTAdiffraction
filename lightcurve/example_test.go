package lightcurve_test

import (
	"fmt"
	"log"

	"github.com/bob-anderson-ok/IOTAdiffraction/lightcurve"
)

// Example demonstrates how to use the lightcurve package to:
// 1. Load a 16-bit diffraction image and extract a light curve
// 2. Load a geometric shadow image and detect edges
// 3. Plot the light curve with edge markers
// 4. Draw the observation path on an 8-bit display image
func Example() {
	// Define the observation path using velocity components and offset
	// These parameters come from the occultation event parameter file
	path := &lightcurve.ObservationPath{
		// Shadow velocity components (from the parameter file)
		DxKmPerSec:             -15.0, // Shadow velocity X component (km/sec)
		DyKmPerSec:             -10.0, // Shadow velocity Y component (km/sec)
		PathOffsetFromCenterKm: 20.0,  // Perpendicular offset from the center (km)

		// Fundamental plane dimensions
		FundamentalPlaneWidthKm:  500.0, // Width of the fundamental plane in km
		FundamentalPlaneWidthPts: 1000,  // Width of the fundamental plane in pixels
	}

	// Compute the path start/end points from velocity and offset
	err := path.ComputePathFromVelocity()
	if err != nil {
		log.Fatalf("Failed to compute path: %v", err)
	}

	fmt.Printf("Shadow speed: %.3f km/sec\n", path.ShadowSpeedKmPerSec)
	fmt.Printf("Path angle: %.1f degrees\n", path.PathAngleDegrees)
	fmt.Printf("Path direction: %s\n", path.Direction)
	fmt.Printf("Path start: (%.1f, %.1f)\n", path.StartX, path.StartY)
	fmt.Printf("Path end: (%.1f, %.1f)\n", path.EndX, path.EndY)

	// Compute sample points along the path
	path.ComputeSamplePoints()
	fmt.Printf("Generated %d sample points along the observation path\n", len(path.SamplePoints))

	// Load the 16-bit diffraction image
	// The scale factor of 4000 matches what the main application uses
	intensityMatrix, err := lightcurve.LoadGray16PNG("occultImage16bit.png", 4000.0)
	if err != nil {
		log.Printf("Could not load 16-bit image: %v\n", err)
		// For an example, we'll create a simple test matrix
		intensityMatrix = createTestIntensityMatrix(1000)
	}

	// Extract the light curve from the intensity matrix
	lightCurveData := lightcurve.ExtractLightCurve(intensityMatrix, path)
	fmt.Printf("Extracted %d light curve points\n", len(lightCurveData))

	// Print the first few points
	fmt.Println("\nFirst 5 light curve points:")
	for i := 0; i < 5 && i < len(lightCurveData); i++ {
		fmt.Printf("  Distance: %.3f km, Intensity: %.4f\n",
			lightCurveData[i].Distance, lightCurveData[i].Intensity)
	}

	// Load the geometric shadow image and detect edges
	geometricMatrix, err := lightcurve.LoadGray8PNG("geometricShadow.png")
	if err != nil {
		log.Printf("Could not load geometric shadow: %v\n", err)
		// For an example, we'll create a simple test matrix
		geometricMatrix = createTestGeometricMatrix(1000)
	}

	// Find edges in the geometric shadow
	edges := lightcurve.FindEdgesInGeometricShadow(geometricMatrix, path)
	fmt.Printf("\nFound %d edges in the geometric shadow\n", len(edges))
	for i, edge := range edges {
		distanceKm := edge * path.FundamentalPlaneWidthKm / float64(path.FundamentalPlaneWidthPts)
		fmt.Printf("  Edge %d at pixel distance %.1f (%.3f km)\n", i+1, edge, distanceKm)
	}

	// Create and save the light curve plot
	err = lightcurve.SaveLightCurvePlot("lightcurve_plot.png", lightCurveData, edges, path, 1200, 500)
	if err != nil {
		log.Printf("Could not save light curve plot: %v\n", err)
	} else {
		fmt.Println("\nSaved light curve plot to lightcurve_plot.png")
	}

	// Load the 8-bit display image and draw the observation line
	displayImage, err := lightcurve.LoadImageFromFile("diffractionImage8bit.png")
	if err != nil {
		log.Printf("Could not load display image: %v\n", err)
	} else {
		// Draw the observation path on the image
		annotatedImage, err := lightcurve.DrawObservationLineOnImage(displayImage, path)
		if err != nil {
			log.Printf("Could not draw observation line: %v\n", err)
		} else {
			// Save the annotated image
			err = lightcurve.SaveImageToFile("annotated_diffraction.png", annotatedImage)
			if err != nil {
				log.Printf("Could not save annotated image: %v\n", err)
			} else {
				fmt.Println("Saved annotated image to annotated_diffraction.png")
			}
		}
	}

	// Output:
	// Generated 1273 sample points along the observation path
	// Extracted 1273 light curve points
}

// createTestIntensityMatrix creates a simple test intensity matrix with a diffraction-like pattern
func createTestIntensityMatrix(size int) [][]float64 {
	matrix := make([][]float64, size)
	center := float64(size) / 2.0

	for y := 0; y < size; y++ {
		matrix[y] = make([]float64, size)
		for x := 0; x < size; x++ {
			// Simple circular pattern
			dx := float64(x) - center
			dy := float64(y) - center
			r := (dx*dx + dy*dy) / (center * center)

			// Create a simple diffraction-like pattern
			if r < 0.1 {
				matrix[y][x] = 0.0 // Shadow
			} else if r < 0.15 {
				matrix[y][x] = 1.2 // Bright ring (diffraction peak)
			} else {
				matrix[y][x] = 1.0 // Normal intensity
			}
		}
	}
	return matrix
}

// createTestGeometricMatrix creates a simple test geometric shadow matrix
func createTestGeometricMatrix(size int) [][]float64 {
	matrix := make([][]float64, size)
	center := float64(size) / 2.0
	radius := float64(size) * 0.15

	for y := 0; y < size; y++ {
		matrix[y] = make([]float64, size)
		for x := 0; x < size; x++ {
			dx := float64(x) - center
			dy := float64(y) - center
			r := dx*dx + dy*dy

			if r < radius*radius {
				matrix[y][x] = 0.0 // Inside the occulter (shadow)
			} else {
				matrix[y][x] = 1.0 // Outside (illuminated)
			}
		}
	}
	return matrix
}
