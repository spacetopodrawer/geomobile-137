package photogrammetry

import "image"

// ReconstructionPoint represents a 3D point reconstructed from images
type ReconstructionPoint struct {
	X, Y, Z float64 // 3D coordinates (world frame)
	Color   [3]uint8 // RGB color from source image
	ViewCount int    // Number of views observing this point
}

// CameraFrame represents camera pose and parameters
type CameraFrame struct {
	ImageID       int
	Image         image.Image
	RotationMatrix [3][3]float64 // R matrix
	Translation    [3]float64    // T vector
	CameraMatrix   [3][3]float64 // Intrinsics
	IsKeyFrame    bool
}

// SfMIncremental implements incremental Structure-from-Motion
// Week 2-3: Builds 3D model incrementally from image sequence
type SfMIncremental struct {
	ImageSequence []*CameraFrame
	Points3D      []*ReconstructionPoint
	FeatureDetector *FeatureDetector
	FeatureMatcher  *FeatureMatcher
}

// NewSfMIncremental creates a new incremental SfM reconstructor
func NewSfMIncremental() *SfMIncremental {
	return &SfMIncremental{
		ImageSequence:   make([]*CameraFrame, 0),
		Points3D:        make([]*ReconstructionPoint, 0),
		FeatureDetector: NewFeatureDetector(),
		FeatureMatcher:  NewFeatureMatcher(),
	}
}

// AddImage adds an image to the reconstruction sequence
func (s *SfMIncremental) AddImage(img image.Image) error {
	// TODO: Week 2-3 implementation
	// 1. Detect features in image
	// 2. Match with previous frames
	// 3. Estimate camera pose (if not first frame)
	// 4. Triangulate new points
	// 5. Refine with bundle adjustment
	return nil
}

// GetReconstruction returns the current 3D point cloud
func (s *SfMIncremental) GetReconstruction() []*ReconstructionPoint {
	return s.Points3D
}

// GetCameraPoses returns estimated camera poses for all images
func (s *SfMIncremental) GetCameraPoses() []*CameraFrame {
	return s.ImageSequence
}
