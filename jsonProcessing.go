package main

import json "github.com/KevinWang15/go-json5"

func parseArrayFormat(data []byte) ([][2]float64, error) {
	var pairs [][2]float64
	err := json.Unmarshal(data, &pairs)
	return pairs, err
}

func getLeafValue(jsonTable map[string]interface{}, path ...string) (interface{}, bool) {
	var cur interface{} = jsonTable
	for _, p := range path {
		m, ok := cur.(map[string]interface{})
		if !ok {
			return nil, false
		}
		cur, ok = m[p]
		if !ok {
			return nil, false
		}
	}
	return cur, true
}

func validateJsonFileAndFillEvent(jsonTable map[string]interface{}, event *OccultationEvent) (string, bool) {
	msg := "No problem found in json file" // Initialize msg to presumed success.

	showInput, ok := getLeafValue(jsonTable, "show_input_bool")
	if !ok {
		event.ShowInput = false // default to false if this field is missing
	} else {
		event.ShowInput, ok = showInput.(bool)
		if !ok {
			msg = "show_input_bool: is not a bool"
			return msg, false
		}
	}

	//rotationFlag, ok := getLeafValue(jsonTable, "rotate_ground_shadow_to_90_degree_pa_bool")
	//if !ok {
	//	event.RotateGroundShadowTo90pa = true // Default: rotate ground shadow to a standard 90 degree PA
	//} else {
	//	flagValue, ok := rotationFlag.(bool)
	//	if !ok {
	//		msg = "rotate_ground_shadow_to_90_degree_pa_bool: is not a bool"
	//		return msg, false
	//	}
	//	event.RotateGroundShadowTo90pa = flagValue
	//}

	windowSize, ok := getLeafValue(jsonTable, "window_size_pixels")
	if !ok {
		event.WindowSizePixels = 500 // Default to 500 pixels if this field is missing
	} else {
		wSize, ok := windowSize.(float64)
		if !ok {
			msg = "window_size_pixels: is not a float64"
			return msg, false
		}
		event.WindowSizePixels = int(wSize)
	}

	//filePath, ok = getLeafValue(jsonTable, "path_for_ground_shadow_output_folder")
	//if !ok {
	//	msg = "path_for_ground_shadow_output_folder: not found"
	//	return msg, false
	//}
	//event.PathForGroundShadowOutputFolder, ok = filePath.(string)
	//if !ok {
	//	msg = "path_for_ground_shadow_output_folder: is not a string"
	//	return msg, false
	//}

	filePath, ok := getLeafValue(jsonTable, "path_to_qe_table_file")
	if ok {
		event.PathToQEtable, ok = filePath.(string)
		if !ok {
			msg = "path_to_qe_table_file: is not a string"
			return msg, false
		}
	}

	mainBodyRequired := true
	filePath, ok = getLeafValue(jsonTable, "path_to_external_image")
	if ok {
		event.PathToExternalImage, ok = filePath.(string)
		if !ok {
			msg = "path_to_external_image: is not a string"
			return msg, false
		}
		mainBodyRequired = false
	}

	title, ok := getLeafValue(jsonTable, "title")
	if ok {
		event.Title, ok = title.(string)
		if !ok {
			msg = "title: is not a string"
			return msg, false
		}
	}

	pathOffsetKm, ok := getLeafValue(jsonTable, "path_perpendicular_offset_from_center_km")
	if ok { // We allow this field to be missing - if missing, it defaults to 0
		event.PathOffsetFromCenterKm, ok = pathOffsetKm.(float64)
		if !ok {
			msg = "path_perpendicular_offset_from_center_km: is not a float64"
			return msg, false
		}
	}

	skyWidth, ok := getLeafValue(jsonTable, "fundamental_plane_width_km")
	if !ok {
		msg = "fundamental_plane_width_km: not found"
		return msg, false
	}
	event.FundamentalPlaneWidthKm, ok = skyWidth.(float64)
	if !ok {
		msg = "fundamental_plane_width_km: is not a float64"
		return msg, false
	}

	numPts, ok := getLeafValue(jsonTable, "fundamental_plane_width_num_points")
	if !ok {
		msg = "fundamental_plane_width_num_points: not found"
		return msg, false
	}
	numberOfPoints, ok := numPts.(float64)
	if !ok {
		msg = "fundamental_plane_width_num_points: is not a float64"
		return msg, false
	}
	event.FundamentalPlaneWidthPoints = int(numberOfPoints)

	//expSecs, ok := getLeafValue(jsonTable, "camera_exposure_secs")
	//if !ok {
	//	msg = "camera_exposure_secs: not found"
	//	return msg, false
	//}
	//event.CameraExposureSecs, ok = expSecs.(float64)
	//if !ok {
	//	msg = "camera_exposure_secs: is not a float64"
	//	return msg, false
	//}

	magDropPercent, ok := getLeafValue(jsonTable, "percent_mag_drop")
	if ok {
		event.PercentMagDrop, ok = magDropPercent.(float64)
		if !ok {
			msg = "percent_mag_drop: is not a float64"
			return msg, false
		}
	}

	wavelength, ok := getLeafValue(jsonTable, "observation_wavelength_nm")
	if !ok {
		msg = "observation_wavelength_nm: not found"
		return msg, false
	}
	event.ObservationWavelengthNm, ok = wavelength.(float64)
	if !ok {
		msg = "observation_wavelength_nm: is not a float64"
		return msg, false
	}

	dX, ok := getLeafValue(jsonTable, "dX_km_per_sec")
	if ok {
		event.DxKmPerSec, ok = dX.(float64)
		if !ok {
			msg = "dX_km_per_sec: is not a float64"
			return msg, false
		}
	}

	dY, ok := getLeafValue(jsonTable, "dY_km_per_sec")
	if ok {
		event.DyKmPerSec, ok = dY.(float64)
		if !ok {
			msg = "dY_km_per_sec: is not a float64"
			return msg, false
		}
	}

	starName, ok := getLeafValue(jsonTable, "star_name")
	if ok {
		event.StarName, ok = starName.(string)
		if !ok {
			msg = "star_name: is not a string"
			return msg, false
		}
	}

	starDiam, ok := getLeafValue(jsonTable, "star_diam_on_plane_mas")
	if !ok {
		event.StarDiamMas = 0.0 // Default value
	} else {
		event.StarDiamMas, ok = starDiam.(float64)
		if !ok {
			msg = "star_diam_on_plane_mas: is not a float64"
			return msg, false
		}
	}

	limbCoeff, ok := getLeafValue(jsonTable, "limb_darkening_coeff")
	if !ok {
		event.LimbDarkeningCoeff = 0.0 // Default value
	} else {
		event.LimbDarkeningCoeff, ok = limbCoeff.(float64)
		if !ok {
			msg = "limb_darkening_coeff: is not a float64"
			return msg, false
		}
	}

	starClass, ok := getLeafValue(jsonTable, "star_class")
	if !ok {
		event.StarClass = "" // Default value
	} else {
		event.StarClass, ok = starClass.(string)
		if !ok {
			msg = "star_class: is not a string"
			return msg, false
		}
	}

	needAdistanceMeasure := true
	parallax, ok := getLeafValue(jsonTable, "parallax_arcsec")
	if ok {
		event.ParallaxArcsec, ok = parallax.(float64)
		if !ok {
			msg = "parallax_arcsec: is not a float64"
			return msg, false
		}
		needAdistanceMeasure = false // Now we don't need distance_au'
	}

	distanceAU, ok := getLeafValue(jsonTable, "distance_au")
	if !ok {
		if needAdistanceMeasure {
			msg = "distance_au: not found"
			return msg, false
		}
	} else {
		event.DistanceAu, ok = distanceAU.(float64)
		if !ok {
			msg = "distance_au: is not a float64"
			return msg, false
		}
	}

	// Check to see if a main_body group is present. Required if no external image is supplied.
	_, ok = getLeafValue(jsonTable, "main_body")
	event.MainBodyGiven = ok

	if ok {
		// Validate the main_body.x_center_km entry
		v, ok := getLeafValue(jsonTable, "main_body", "x_center_km")
		if ok {
			value, ok := v.(float64)
			if ok {
				event.MainBodyXCenterKm = value
			} else {
				msg = "main_body.x_center_km: is not a float64"
				return msg, false
			}
		} else {
			msg = "main_body.x_center_km: not found"
			return msg, false
		}

		// Validate the main_body.y_center_km entry
		v, ok = getLeafValue(jsonTable, "main_body", "y_center_km")
		if ok {
			value, ok := v.(float64)
			if ok {
				event.MainBodyYCenterKm = value
			} else {
				msg = "main_body.y_center_km: is not a float64"
				return msg, false
			}
		} else {
			msg = "main_body.y_center_km: not found"
			return msg, false
		}

		// Validate the main_body.major_axis_km entry
		v, ok = getLeafValue(jsonTable, "main_body", "major_axis_km")
		if ok {
			value, ok := v.(float64)
			if ok {
				event.MainbodyMajorAxisKm = value
			} else {
				msg = "main_body.major_axis_km: is not a float64"
				return msg, false
			}
		} else {
			msg = "main_body.major_axis_km: not found"
			return msg, false
		}

		// Validate the main_body.minor_axis_km entry
		v, ok = getLeafValue(jsonTable, "main_body", "minor_axis_km")
		if ok {
			value, ok := v.(float64)
			if ok {
				event.MainbodyMinorAxisKm = value
			} else {
				msg = "main_body.minor_axis_km: is not a float64"
				return msg, false
			}
		} else {
			msg = "main_body.minor_axis_km: not found"
			return msg, false
		}

		// Validate the main_body.major_axis_pa_degrees entry
		v, ok = getLeafValue(jsonTable, "main_body", "major_axis_pa_degrees")
		if ok {
			value, ok := v.(float64)
			if ok {
				event.MainbodyMajorAxisPaDegrees = value
			} else {
				msg = "main_body.major_axis_pa_degrees: is not a float64"
				return msg, false
			}
		} else {
			msg = "main_body.major_axis_pa_degrees: not found"
			return msg, false
		}
	} else {
		if mainBodyRequired {
			msg = "main_body group not found and is required."
			return msg, false
		}
	}

	// Check to see if a satellite group is present --- it is optional
	_, ok = getLeafValue(jsonTable, "satellite")
	event.SatelliteGiven = ok

	if ok {
		// Validate the satellite.x_center_km entry
		v, ok := getLeafValue(jsonTable, "satellite", "x_center_km")
		if ok {
			value, ok := v.(float64)
			if ok {
				event.SatelliteXCenterKm = value
			} else {
				msg = "satellite.x_center_km: is not a float64"
				return msg, false
			}
		} else {
			msg = "satellite.x_center_km: not found"
			return msg, false
		}

		// Validate the satellite.y_center_km entry
		v, ok = getLeafValue(jsonTable, "satellite", "y_center_km")
		if ok {
			value, ok := v.(float64)
			if ok {
				event.SatelliteYCenterKm = value
			} else {
				msg = "satellite.y_center_km: is not a float64"
				return msg, false
			}
		} else {
			msg = "satellite.y_center_km: not found"
			return msg, false
		}

		// Validate the satellite.major_axis_km entry
		v, ok = getLeafValue(jsonTable, "satellite", "major_axis_km")
		if ok {
			value, ok := v.(float64)
			if ok {
				event.SatelliteMajorAxisKm = value
			} else {
				msg = "satellite.major_axis_km: is not a float64"
				return msg, false
			}
		} else {
			msg = "satellite.major_axis_km: not found"
			return msg, false
		}

		// Validate the satellite.minor_axis_km entry
		v, ok = getLeafValue(jsonTable, "satellite", "minor_axis_km")
		if ok {
			value, ok := v.(float64)
			if ok {
				event.SatelliteMinorAxisKm = value
			} else {
				msg = "satellite.minor_axis_km: is not a float64"
				return msg, false
			}
		} else {
			msg = "satellite.minor_axis_km: not found"
			return msg, false
		}

		// Validate the satellite.major_axis_pa_degrees entry
		v, ok = getLeafValue(jsonTable, "satellite", "major_axis_pa_degrees")
		if ok {
			value, ok := v.(float64)
			if ok {
				event.SatelliteMajorAxisPaDegrees = value
			} else {
				msg = "satellite.major_axis_pa_degrees: is not a float64"
				return msg, false
			}
		} else {
			msg = "satellite.major_axis_pa_degrees: not found"
			return msg, false
		}
	}

	return msg, true
}
