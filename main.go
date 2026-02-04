package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	json "github.com/KevinWang15/go-json5"
)

// !!!!! This MUST match the app name given in the run configuration !!!!!
const version = "1_5_1"

// !!!!! This MUST match the app name given in the run configuration !!!!!

type OccultationEvent struct {
	FplaneImage                     *image.Gray // A square array of uint8 values
	IntensityMatrix                 [][]float64
	GeometricMatrix                 [][]float64
	ShowInput                       bool
	PathSamplePoints                [][3]float64
	PathStart                       [2]float64 // [x,y] fractional pixel coordinates of path start point
	PathEnd                         [2]float64 // [x,y] fractional pixel coordinates of path end point
	PathDirection                   string
	WindowSizePixels                int
	PathForGroundShadowOutputFolder string
	PathToExternalImage             string
	PathToQEtable                   string
	QEtable                         [][2]float64
	Title                           string
	FundamentalPlaneWidthKm         float64
	FundamentalPlaneWidthPoints     int
	ObservationWavelengthNm         float64
	DxKmPerSec                      float64
	DyKmPerSec                      float64
	ShadowSpeedKmPerSec             float64
	PathAngleDegrees                float64
	PathOffsetFromCenterKm          float64
	StarName                        string
	StarDiamMas                     float64
	StarDiamKm                      float64
	LimbDarkeningCoeff              float64
	StarClass                       string
	PercentMagDrop                  float64
	ParallaxArcsec                  float64
	DistanceAu                      float64
	MainBodyGiven                   bool
	MainBodyXCenterKm               float64
	MainBodyYCenterKm               float64
	MainbodyMajorAxisKm             float64
	MainbodyMinorAxisKm             float64
	MainbodyMajorAxisPaDegrees      float64
	SatelliteGiven                  bool
	SatelliteXCenterKm              float64
	SatelliteYCenterKm              float64
	SatelliteMajorAxisKm            float64
	SatelliteMinorAxisKm            float64
	SatelliteMajorAxisPaDegrees     float64
}

func main() {

	programStart := time.Now()

	var p1 AnnotatedPoint
	var p2 AnnotatedPoint

	// We supply an ID (hopefully unique) because we may need to use the preferences API
	myApp := app.NewWithID("com.gmail.ok.anderson.bob")
	w := myApp.NewWindow("OccultDiffractionApp - user friendly diffraction image (8 bit grayscale png)")
	w.Resize(fyne.Size{Height: 800, Width: 1200})

	args := os.Args

	if len(args) != 2 {
		fmt.Println("\n\tWrong number of arguments.\n\tUsage: OccultDiffractionApp <parameter-file>")
		os.Exit(1)
	}

	path := args[1]

	// Read the Json5 (or Json) parameter file
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Println(fmt.Errorf("\n\tAttempt to read input file %q failed: %w\n", path, err))
		os.Exit(2)
	}

	// Parse json(5) data into a generic container
	var jsonTable map[string]interface{}
	err = json.Unmarshal(data, &jsonTable)
	if err != nil {
		fmt.Println(fmt.Errorf("\n\tFormat error in file %q: %w\n", path, err))
		os.Exit(3)
	}

	var event OccultationEvent
	msg, ok := validateJsonFileAndFillEvent(jsonTable, &event)
	if !ok {
		fmt.Println(msg)
		os.Exit(4)
	}

	// Check for user wanting printout of complete jsonTable
	if event.ShowInput {
		fmt.Printf("%s", "\nPrintout of  complete jsonTable contents...\n")
		fmt.Println(string(data))
	}

	// If a path to a camera response json file was given, read it
	if event.PathToQEtable != "" {
		// Read the Json5 (or Json) parameter file
		data, err := os.ReadFile(event.PathToQEtable)
		if err != nil {
			fmt.Println(fmt.Errorf("\n\tAttempt to read file %q failed: %w\n", path, err))
			os.Exit(13)
		}
		var qeTable [][2]float64
		qeTable, err = parseArrayFormat(data)
		if err != nil {
			fmt.Println(fmt.Errorf("\n\tError reading camera response file %q: %w\n", event.PathToQEtable, err))
			os.Exit(15)
		}
		event.QEtable = qeTable
		//fmt.Println("Got the camera table", len(qeTable), "entries")
		if len(qeTable) < 1 {
			fmt.Println(fmt.Errorf("\n\tThe camera response file %q is empty.", event.PathToQEtable))
			os.Exit(14)
		}
		var cumWeights = 0.0
		for i := 0; i < len(qeTable); i++ {
			cumWeights += qeTable[i][1]
		}
		for i := 0; i < len(qeTable); i++ {
			qeTable[i][1] /= cumWeights
		}
		MakeCameraResponsePlot(qeTable, event.PathToQEtable)
	}

	// Sanity check on number of points in a fundamental plane
	if event.FundamentalPlaneWidthPoints < 10 {
		fmt.Println(fmt.Errorf("\n\tThe fundamental plane width must be at least 10 points."))
		os.Exit(16)
	}

	Npts := event.FundamentalPlaneWidthPoints // Just a shorthand version

	fmt.Printf("\nVersion %s\n\n", version)

	// Calculate resolution in fundamental plane
	resolution := event.FundamentalPlaneWidthKm / float64(Npts)
	fmt.Printf("Resolution in fundamental plane is %0.3f km/pixel\n", resolution)
	fresnelScale := FresnelScale(event.ObservationWavelengthNm, event.DistanceAu)
	fmt.Printf("Fresnel scale is %0.3f km\n", fresnelScale)
	samplesPerFresnelScale := int(fresnelScale / resolution)
	fmt.Printf("Samples per Fresnel scale is %d  (To see diffraction effects, this number should be at least 5)\n\n", samplesPerFresnelScale)

	start := time.Now() // Time generation of geometric shadow

	// Deal with external image supplied by the user.
	if event.PathToExternalImage != "" {
		f, err := os.Open(event.PathToExternalImage)
		if err != nil {
			fmt.Println(fmt.Errorf("\n\tAttempt to read external image %q failed: %w\n", event.PathToExternalImage, err))
			os.Exit(5)
		}
		//defer f.Close()
		defer func() {
			if cerr := f.Close(); cerr != nil && err == nil {
				err = cerr
			}
		}()

		img, err := png.Decode(f)
		if err != nil {
			fmt.Println(fmt.Errorf("\n\tAttempt to decode external image %q failed: %w\n", event.PathToExternalImage, err))
			os.Exit(6)
		}

		if img.Bounds().Dx() != img.Bounds().Dy() {
			fmt.Println(fmt.Errorf("\n\tThe supplied external image %q is not square.", event.PathToExternalImage))
			os.Exit(7)
		}

		// We require that an external image is supplied in GRAY format (uint8) to match
		// our internal use when we build the fundamental plane image ourselves. We do this
		// so that we can add (overlay) any ellipses defined in the json file. We expect
		// that external image files are used only to define odd or polygon shapes.
		if img.ColorModel() != color.GrayModel {
			fmt.Println(fmt.Errorf("\n\tThe supplied external image %q is not type GRAY.", event.PathToExternalImage))
			os.Exit(8)
		}

		event.FplaneImage = img.(*image.Gray)

		// Override the value (possibly) supplied in the fundamental_plane_width_num_points parameter
		event.FundamentalPlaneWidthPoints = img.Bounds().Dx()
		Npts = event.FundamentalPlaneWidthPoints // Shorthand
		fmt.Println(ColorModelString(img.ColorModel()))
	} else { // No image supplied by user, so we start our own.
		event.FplaneImage = image.NewGray(image.Rect(0, 0, Npts, Npts))
		FillFplane(event.FplaneImage, true)
	}

	AddEllipses(event, true)
	err = SaveGrayPNG("geometricShadow.png", event.FplaneImage)
	if err != nil {
		fmt.Println(fmt.Errorf("\n\tFailed to write %q.", "geometricShadow.png"))
		os.Exit(9)
	}

	sourcePlane := ConvertSourcePlaneImageToComplex(event.FplaneImage)
	event.GeometricMatrix = ConvertSourcePlaneImageToMatrix(event.FplaneImage)

	elapsed := time.Since(start)
	fmt.Printf("Generation of the geometric shadow took %s\n", elapsed)

	// Here we figure out the proper value to use for the limb darkening coefficient based on
	// the supplied parameters.
	LimbValues := map[string]float64{
		"O": 0.05,
		"B": 0.2,
		"A": 0.5,
		"F": 0.6,
		"G": 0.7,
		"K": 0.7,
		"M": 0.7,
	}
	if event.StarDiamMas > 0.0 {
		if event.LimbDarkeningCoeff == 0.0 { // Limb darkening coefficient takes precedence over star class
			if event.StarClass == "" {
				// No star class or limb darkening coefficient given, so we use a default value of 0.7
				event.LimbDarkeningCoeff = 0.7
			} else {
				v, ok := LimbValues[event.StarClass]
				if !ok {
					fmt.Println(fmt.Errorf(
						"\n\tThe star class %q is not recognized. Default value of 0.7 will be used.\n",
						event.StarClass),
					)
					event.LimbDarkeningCoeff = 0.7
				} else {
					event.LimbDarkeningCoeff = v // Use value from the table
				}
			}
		}
	}

	fmt.Println("Limb darkening coefficient set to:", event.LimbDarkeningCoeff)

	// If a user gave us distance in arcseconds, it is given priority, and
	// we overwrite any value that may also have been given in AU.
	if event.ParallaxArcsec > 0.0 {
		event.DistanceAu = 8.79414 / event.ParallaxArcsec
	}

	auToKm := 1.495979e+8
	nmToKm := 1e-9 * 1e-3

	WavelengthKm := event.ObservationWavelengthNm * nmToKm
	Lkm := event.FundamentalPlaneWidthKm
	Zkm := event.DistanceAu * auToKm

	// Some elementary checks to make sure that the user has not supplied bad parameters
	if Lkm <= 0.0 {
		fmt.Println(fmt.Errorf("\n\tFundamental plane width must be positive."))
		os.Exit(10)
	}

	if Zkm <= 0.0 {
		fmt.Println(fmt.Errorf("\n\tDistance given is invalid."))
		os.Exit(10)
	}

	event.ShadowSpeedKmPerSec = math.Sqrt(event.DxKmPerSec*event.DxKmPerSec + event.DyKmPerSec*event.DyKmPerSec)
	if event.ShadowSpeedKmPerSec > 0.0 {
		event.PathAngleDegrees = math.Atan2(-event.DxKmPerSec, -event.DyKmPerSec) * 180.0 / math.Pi
		if event.PathAngleDegrees < 0.0 {
			event.PathAngleDegrees += 360.0
		}
		fmt.Printf("\nPath angle is %0.1f degrees\n", event.PathAngleDegrees)
		fmt.Printf("Shadow speed is %0.3f km/sec\n\n", event.ShadowSpeedKmPerSec)

		// The following function sets event.PathStart and event.PathEnd variables
		p1, p2, event.PathDirection, err = processPathDirection(Npts, p1, p2, &event)
		if err != nil {
			fmt.Println(fmt.Errorf("\n\tProcessing of path direction failed: %w", err))
			os.Exit(10)
		}
		fmt.Printf("Direction: %s\n", event.PathDirection)
		computePathPoints(&event)

	}

	event.StarDiamKm = 1.496e8 * event.DistanceAu * event.StarDiamMas / (1000.0 * 206265)

	var eField []complex128
	if len(event.QEtable) > 0 {
		// Get the first scaled eField to use to accumulate all the rest
		WavelengthKm = event.QEtable[0][0] * nmToKm
		eField = FullObservationPlaneSincSolution(Lkm, Zkm, WavelengthKm, sourcePlane)
		scaleComplex(eField, event.QEtable[0][1])

		// Now do the rest
		for i := 1; i < len(event.QEtable); i++ {
			// Compute the effective wavelength at each wavelength bin
			WavelengthKm = event.QEtable[i][0] * nmToKm
			start = time.Now()
			newField := FullObservationPlaneSincSolution(Lkm, Zkm, WavelengthKm, sourcePlane)
			addScaledComplexInPlace(eField, newField, event.QEtable[i][1])
			elapsed = time.Since(start)
			fmt.Printf("Calculation of wavelength %0.1f e-field took %s\n", event.QEtable[i][0], elapsed)
		}
	} else {
		start = time.Now()
		eField = FullObservationPlaneSincSolution(Lkm, Zkm, WavelengthKm, sourcePlane)
		elapsed = time.Since(start)
		fmt.Printf("Calculation of the observation e-field took %s\n", elapsed)
	}

	start = time.Now()

	// incidentWave is used to convert the aperture image to an occulter image using Babinet's formula
	incidentWave := complex(1.0, 0.0)

	intensity := make([]float64, len(eField))
	for i := 0; i < len(eField); i++ {
		intensity[i] = real(incidentWave-eField[i])*real(incidentWave-eField[i]) +
			imag(incidentWave-eField[i])*imag(incidentWave-eField[i])
	}

	event.IntensityMatrix, err = Reshape1DTo2D(intensity, Npts, Npts)
	if err != nil {
		fmt.Println(fmt.Errorf("reshape of intensity vector failed: %w", err))
		os.Exit(10)
	}

	// Here we apply any necessary magDrop adjustments
	if event.PercentMagDrop > 0 { // Check for value given and bonus: ignore negative values
		if event.PercentMagDrop > 100 {
			fmt.Println(fmt.Errorf("percentMagDrop of %0.1f is too large. Setting it to 100.0", event.PercentMagDrop))
			event.PercentMagDrop = 100.0
		}
		scaleFactor := event.PercentMagDrop / 100.0
		shiftUp := 1.0 - scaleFactor
		for row := 0; row < len(event.IntensityMatrix); row++ {
			for col := 0; col < len(event.IntensityMatrix[row]); col++ {
				event.IntensityMatrix[row][col] *= scaleFactor
				event.IntensityMatrix[row][col] += shiftUp
			}
		}
	}

	elapsed = time.Since(start)
	fmt.Printf("Calculation of the observation intensity took %s\n", elapsed)

	var newImage [][]float64

	if event.StarDiamKm > 0.0 {
		fmt.Printf("\nStar diameter projected at the plane of the asteroid is %0.3f km\n\n", event.StarDiamKm)
		starImage, sumOfWeights := BuildStarPsf(event.StarDiamKm, resolution, event.LimbDarkeningCoeff)

		start := time.Now()
		newImage, err = ConvolvePSFFFT(event.IntensityMatrix, starImage, sumOfWeights, ConvSame, PadReplicate, false)
		if err != nil {
			fmt.Println(fmt.Errorf("convolution of intensity matrix with star image failed: %w", err))
			os.Exit(13)
		}

		event.IntensityMatrix = newImage

		elapsed := time.Since(start)
		fmt.Printf("Convolution of intensity matrix with star image took %s\n", elapsed)

		imgForDisplay, err := MatrixToGrayViewPercentile(newImage, 0.0, 100)
		// comment place here just to suppress dup lines warning
		if err != nil {
			fmt.Println(fmt.Errorf("creation of the display image failed: %w", err))
			os.Exit(11)
		}

		err = SaveGrayPNG("diffractionImage8bit.png", imgForDisplay)
		if err != nil {
			fmt.Println(fmt.Errorf("writing of %q failed: %w", "diffractionImage8bit.png", err))
			os.Exit(12)
		}

		// Make the scientific (well-defined scaling) version of the intensity matrix
		occultImage, err := MatrixToGray16Data(newImage, 4000)
		if err != nil {
			fmt.Println(fmt.Errorf("creation of occultImage failed: %w", err))
			os.Exit(13)
		}

		err = SaveGray16PNG("occultImage16bit.png", occultImage)
		if err != nil {
			fmt.Println(fmt.Errorf("writing of %q failed: %w", "occultImage16bit.png", err))
			os.Exit(14)
		}
	} else {
		// Make a user-friendly .png of the observation intensity matrix
		imgForDisplay, err := MatrixToGrayViewPercentile(event.IntensityMatrix, 0.0, 100)
		if err != nil {
			fmt.Println(fmt.Errorf("creation of the display image failed: %w", err))
			os.Exit(11)
		}

		err = SaveGrayPNG("diffractionImage8bit.png", imgForDisplay)
		if err != nil {
			fmt.Println(fmt.Errorf("writing of %q failed: %w", "diffractionImage8bit.png", err))
			os.Exit(12)
		}

		// Make the scientific (well-defined scaling) version of the intensity matrix
		occultImage, err := MatrixToGray16Data(event.IntensityMatrix, 4000)
		if err != nil {
			fmt.Println(fmt.Errorf("creation of occultImage failed: %w", err))
			os.Exit(13)
		}

		err = SaveGray16PNG("occultImage16bit.png", occultImage)
		if err != nil {
			fmt.Println(fmt.Errorf("writing of %q failed: %w", "occultImage16bit.png", err))
			os.Exit(14)
		}
	}

	//if event.StarDiamKm > 0.0 {
	//	fmt.Printf("\nStar diameter projected at the plane of the asteroid is %0.3f km\n\n", event.StarDiamKm)
	//	starImage, sumOfWeights := BuildStarPsf(event.StarDiamKm, resolution, event.LimbDarkeningCoeff)
	//
	//	start := time.Now()
	//	newImage, err = ConvolvePSFFFT(event.IntensityMatrix, starImage, sumOfWeights, ConvSame, PadReplicate, false)
	//	if err != nil {
	//		fmt.Println(fmt.Errorf("convolution of intensity matrix with star image failed: %w", err))
	//		os.Exit(13)
	//	}
	//
	//	elapsed := time.Since(start)
	//	fmt.Printf("Convolution of intensity matrix with star image took %s\n", elapsed)
	//
	//	imgForDisplay, err := MatrixToGrayViewPercentile(newImage, 0.0, 100)
	//	// comment place here just to suppress dup lines warning
	//	if err != nil {
	//		fmt.Println(fmt.Errorf("creation of the display image failed: %w", err))
	//		os.Exit(11)
	//	}
	//
	//	err = SaveGrayPNG("diffractionImage8bit.png", imgForDisplay)
	//	if err != nil {
	//		fmt.Println(fmt.Errorf("writing of %q failed: %w", "diffractionImage8bit.png", err))
	//		os.Exit(12)
	//	}
	//
	//	// Make the scientific (well-defined scaling) version of the intensity matrix
	//	occultImage, err := MatrixToGray16Data(newImage, 4000)
	//	if err != nil {
	//		fmt.Println(fmt.Errorf("creation of occultImage failed: %w", err))
	//		os.Exit(13)
	//	}
	//
	//	err = SaveGray16PNG("occultImage16bit.png", occultImage)
	//	if err != nil {
	//		fmt.Println(fmt.Errorf("writing of %q failed: %w", "occultImage16bit.png", err))
	//		os.Exit(14)
	//	}
	//}

	elapsed = time.Since(programStart)
	fmt.Printf("\nTotal program run time is %s\n", elapsed)

	if event.WindowSizePixels > 0 { // We have lots of displays to make!
		size := event.WindowSizePixels

		winTitle := event.Title
		if len(event.QEtable) > 0 {
			winTitle += " (multi-wavelength composite using camera response curve)"
		}

		// w is our main window, created at the beginning of the program
		w.SetTitle(winTitle)
		w.SetPadded(false)
		w.CenterOnScreen()

		img := canvas.NewImageFromFile("diffractionImage8bit.png")

		img.FillMode = canvas.ImageFillContain
		w.Resize(fyne.Size{Height: float32(size), Width: float32(size)})

		w.SetContent(container.NewStack(img))

		// Here we add a red line to show the star path with colored dots at the ends to show direction(red to green)
		if event.ShadowSpeedKmPerSec > 0.0 {
			line := canvas.NewLine(color.RGBA{R: 255, A: 255})
			// Convert row, col values to window coordinates
			scaledY1 := float32(p1.Y) / float32(Npts) * float32(size)
			scaledX1 := float32(p1.X) / float32(Npts) * float32(size)

			scaledY2 := float32(p2.Y) / float32(Npts) * float32(size)
			scaledX2 := float32(p2.X) / float32(Npts) * float32(size)

			line.Position1 = fyne.NewPos(scaledX1, scaledY1)
			line.Position2 = fyne.NewPos(scaledX2, scaledY2)
			line.StrokeWidth = 2

			// Here we use PathStart and PathEnd to place red and green dots at the start and end of the real path

			dotSize := float32(10)
			scaledDotX := float32(event.PathStart[0]) / float32(Npts) * float32(size)
			scaledDotY := float32(event.PathStart[1]) / float32(Npts) * float32(size)
			startDot := placeDotAt(scaledDotX, scaledDotY, dotSize, color.RGBA{R: 255, A: 255})

			scaledDotX = float32(event.PathEnd[0]) / float32(Npts) * float32(size)
			scaledDotY = float32(event.PathEnd[1]) / float32(Npts) * float32(size)
			endDot := placeDotAt(scaledDotX, scaledDotY, dotSize, color.RGBA{G: 255, A: 255})

			content := container.NewWithoutLayout(img, line, startDot, endDot)
			w.SetContent(content)
		} else {
			w.SetContent(container.NewStack(img))
		}
		w.Show()

		var img2 image.Image
		gotCurveToPlot := false
		if event.ShadowSpeedKmPerSec > 0.0 {
			gotCurveToPlot = true
			edges := FindEdgesInGeometricShadow(event)
			img2, err = makePlotImage(event.PathDirection, 1200, 500, event, edges)
			if err != nil {
				panic(err)
			}
		}

		if gotCurveToPlot {
			plotImg := canvas.NewImageFromImage(img2)
			plotImg.FillMode = canvas.ImageFillContain
			plotImg.SetMinSize(fyne.NewSize(1200, 500))

			w2 := myApp.NewWindow("Sample light curve")
			w2.SetContent(container.NewCenter(plotImg))
			w2.Resize(fyne.NewSize(950, 550))
			w2.Show()
		}

		if len(event.QEtable) > 0 {
			cameraImg := canvas.NewImageFromFile("camera_response.png")
			cameraImg.FillMode = canvas.ImageFillContain
			cameraImg.SetMinSize(fyne.NewSize(1200, 500))

			w3 := myApp.NewWindow("Camera response curve")
			w3.SetContent(container.NewCenter(cameraImg))
			w3.Resize(fyne.NewSize(950, 550))
			w3.Show()
		}

		w.ShowAndRun()
	}
}

func FresnelScale(wavelengthNm, ZAu float64) float64 {
	auToKm := 1.495979e+8 // Convert distance expressed in AU to km
	nmToKm := 1e-9 * 1e-3 // Convert nm to km
	wavelengthKm := wavelengthNm * nmToKm
	ZKm := ZAu * auToKm
	return math.Sqrt(wavelengthKm * ZKm / 2)
}

func placeDotAt(x, y, diameter float32, col color.Color) *canvas.Circle {
	dot := canvas.NewCircle(col)
	dot.Resize(fyne.NewSize(diameter, diameter))
	dot.Move(fyne.NewPos(x-diameter/2, y-diameter/2))
	return dot
}

func computePathPoints(e *OccultationEvent) {
	xLengthPixels := e.PathEnd[0] - e.PathStart[0]
	yLengthPixels := e.PathEnd[1] - e.PathStart[1]
	pathLengthPixels := math.Sqrt(xLengthPixels*xLengthPixels + yLengthPixels*yLengthPixels)
	fmt.Printf("Path length is %0.3f pixels\n", pathLengthPixels)
	dYPerStep := yLengthPixels / pathLengthPixels
	dXPerStep := xLengthPixels / pathLengthPixels
	startX := e.PathStart[0]
	startY := e.PathStart[1]
	xVal := 0.0
	yVal := 0.0
	k := 0.0
	distanceFromStart := 0.0
	for i := range int(math.Round(pathLengthPixels)) {
		k = float64(i)
		xVal = startX + k*dXPerStep
		yVal = startY + k*dYPerStep
		// distanceFromStart is the pixel distance from the start of the path. It evaluates to k
		distanceFromStart = math.Sqrt(k*k*dXPerStep*dXPerStep + k*k*dYPerStep*dYPerStep)
		e.PathSamplePoints = append(e.PathSamplePoints, [3]float64{xVal, yVal, distanceFromStart})
	}
	//fmt.Println(e.PathSamplePoints[0], e.PathSamplePoints[len(e.PathSamplePoints)-1])
	//fmt.Println()
}
