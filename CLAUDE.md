# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

OccultDiffraction is a Go application that provides full diffraction analysis for the Occult astronomy software. It simulates and visualizes Fresnel diffraction effects during stellar occultation events by asteroids. Developed by Bob Anderson and Dave Herald.

## Build and Run Commands

```bash
# Build the application (replace X_X_X with current version from main.go line 20)
go build -o OccultDiffractionApp_1_5_1.exe

# Run with parameter file
OccultDiffractionApp_1_5_1.exe parameters
```

**Version management**: The version constant is in `main.go:20`. When updating, change both the constant and the build output filename.

**CGO requirement**: This project uses CGO to link OpenBLAS. The following static libraries must be present in the project root:
- `libopenblas.a` - BLAS acceleration
- `libgomp.a`, `libssp.a`, `libucrt.a` - Runtime dependencies

See `openBLASmatrixMultiply.go` for the CGO linker configuration.

## Testing

No standard Go tests exist. Manual test functions are in `testFuncs.go`:
- `runMatrixMultiplicationTest()` - Validates BLAS matrix multiplication performance
- `runFresnelCephesScalarTest()` - Validates Fresnel integral implementation

## Architecture

### Processing Pipeline

1. **Parameter Loading** (`jsonProcessing.go`) - Parse JSON5 parameter files, validate inputs
2. **Fundamental Plane Construction** (`main.go`, `ellipseFuncs.go`) - Load external image or create from ellipse definitions
3. **Fresnel Diffraction** (`sincDiffraction.go`, `fresnelIntegrals.go`) - Matrix-based sinc method with BLAS acceleration
4. **Star Convolution** (`convolution.go`) - 2D FFT convolution with limb-darkened star PSF
5. **Output Generation** (`imageFuncs.go`, `plotFuncs.go`) - Generate images and light curve plots
6. **GUI Display** (`main.go`) - Fyne-based visualization

### Key Data Structure

`OccultationEvent` struct in `main.go:24-67` holds all parameters for an occultation event including geometry, wavelengths, star properties, and camera response.

### Module Responsibilities

| File | Purpose |
|------|---------|
| `main.go` | Entry point, GUI, orchestration |
| `imageFuncs.go` | Image I/O, Gray8/Gray16 formats, matrix conversions |
| `convolution.go` | 2D FFT convolution, star PSF with limb darkening |
| `jsonProcessing.go` | JSON5 parsing, parameter validation |
| `sincDiffraction.go` | Fresnel diffraction via sinc basis functions |
| `fresnelIntegrals.go` | Cephes polynomial approximations for S(x), C(x) |
| `plotFuncs.go` | Light curve and camera response plotting |
| `pathFuncs.go` | Path calculations, edge detection |
| `ellipseFuncs.go` | Ellipse mathematics for body/satellite shapes |
| `openBLASmatrixMultiply.go` | CGO wrapper for complex matrix multiplication (Zgemm3m) |

## Configuration

Parameter files use JSON5 format (see `parameters` file for a documented example). Key parameters:

- `fundamental_plane_width_km` / `fundamental_plane_width_num_points` - Grid dimensions
- `observation_wavelength_nm` - Single wavelength (or use `path_to_qe_table_file` for multi-wavelength)
- `distance_au` or `parallax_arcsec` - Distance to asteroid (parallax takes precedence if both given)
- `main_body` - Ellipse definition with `x_center_km`, `y_center_km`, `major_axis_km`, `minor_axis_km`, `major_axis_pa_degrees`
- `star_diam_on_plane_mas` - Star angular diameter (0 = point source)
- `star_class` or `limb_darkening_coeff` - Limb darkening model (coeff takes precedence)

## Output Files

- `geometricShadow.png` - 8-bit geometric shadow
- `diffractionImage8bit.png` - 8-bit diffraction pattern (percentile stretched)
- `occultImage16bit.png` - 16-bit scientific data
- `camera_response.png` - QE table plot (if provided)

## Physics Notes

- Fresnel diffraction uses sinc basis function method (Cabillos approach)
- Limb darkening coefficients by spectral class: O=0.05, B=0.2, A=0.5, F/G/K/M=0.7
- Multi-wavelength processing weights each wavelength by camera QE response
- Uses Babinet's principle: intensity = |E_field - incident_wave|²
