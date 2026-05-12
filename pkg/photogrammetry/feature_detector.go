package photogrammetry

import (
	"image"
)

// Feature represents a detected image feature (keypoint)
type Feature struct {
	X          float64 // x-coordinate in pixels
	Y          float64 // y-coordinate in pixels
	Scale      float64 // characteristic scale
	Orientation float64 // dominant orientation in radians
	Descriptor []uint8 // 128 or 256 byte descriptor (SIFT/SURF)
}

// FeatureDetector finds distinctive features in images (SIFT/SURF)
// For Week 2: Structure-from-Motion pipeline
type FeatureDetector struct {
	Method      string // "sift" or "surf"
	MaxFeatures int
	Threshold   float64 // Detection threshold
}

// NewFeatureDetector creates a new feature detector
// method: "sift" (more accurate) or "surf" (faster)
func NewFeatureDetector(method string, maxFeatures int) *FeatureDetector {
	return &FeatureDetector{
		Method:      method,
		MaxFeatures: maxFeatures,
		Threshold:   0.03, // Default SIFT threshold
	}
}

// DetectFeatures finds features in an image
// Returns up to MaxFeatures sorted by strength
func (f *FeatureDetector) DetectFeatures(img image.Image) ([]*Feature, error) {
	// TODO: Week 2 implementation
	// 1. Build scale space pyramid (6 octaves, 5 per octave for SIFT)
	// 2. Detect extrema in difference-of-Gaussians
	// 3. Refine keypoint localization
	// 4. Compute dominant orientation(s)
	// 5. Compute feature descriptors
	return nil, nil
}

// DetectFeaturesRegion finds features in a specific image region
func (f *FeatureDetector) DetectFeaturesRegion(img image.Image, x, y, width, height int) ([]*Feature, error) {
	// TODO: Week 2 implementation
	return nil, nil
}
