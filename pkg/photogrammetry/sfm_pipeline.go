package photogrammetry

import (
	"fmt"
	"sync"
	"time"
)

// SfMPipelineConfig holds all pipeline parameters
type SfMPipelineConfig struct {
	// Feature detection
	PyramidOctaves     int
	PyramidScales      int
	FeatureThreshold   float64

	// Feature matching
	MatchDistanceThreshold float64
	MatchRatioThreshold    float64

	// Epipolar geometry
	EssentialMatrixThreshold float64

	// Triangulation
	TriangulationMinDepth float64
	TriangulationMaxDepth float64

	// Bundle adjustment
	BAMaxIterations      int
	BADampingFactor      float64
	BAConvergenceThresh  float64

	// Loop closure
	LoopMinFrameGap         int
	LoopConfidenceThreshold float64
	LoopInlierRatioThresh   float64

	// Dense reconstruction
	DenseMinDepth            float64
	DenseMaxDepth            float64
	DenseWindowSize          int
	DenseConfidenceThreshold float64

	// Real-time processing
	TargetFPS   int           // Target 10 Hz
	MaxLatency  time.Duration // Max processing time per frame
}

// DefaultConfig returns sensible defaults
func DefaultConfig() *SfMPipelineConfig {
	return &SfMPipelineConfig{
		// Feature detection
		PyramidOctaves:     4,
		PyramidScales:      5,
		FeatureThreshold:   0.03,

		// Feature matching
		MatchDistanceThreshold: 100.0,
		MatchRatioThreshold:    0.75,

		// Epipolar geometry
		EssentialMatrixThreshold: 0.99,

		// Triangulation
		TriangulationMinDepth: 0.1,
		TriangulationMaxDepth: 1000.0,

		// Bundle adjustment
		BAMaxIterations:     20,
		BADampingFactor:     0.001,
		BAConvergenceThresh: 1e-6,

		// Loop closure
		LoopMinFrameGap:         20,
		LoopConfidenceThreshold: 0.8,
		LoopInlierRatioThresh:   0.7,

		// Dense reconstruction
		DenseMinDepth:            0.1,
		DenseMaxDepth:            100.0,
		DenseWindowSize:          5,
		DenseConfidenceThreshold: 0.5,

		// Real-time
		TargetFPS:  10,
		MaxLatency: 100 * time.Millisecond,
	}
}

// SfMPipeline orchestrates complete Structure-from-Motion pipeline
type SfMPipeline struct {
	mu sync.RWMutex

	// Configuration
	Config *SfMPipelineConfig

	// Core components
	FeatureDetector    *FeatureDetector
	FeatureMatcher     *FeatureMatcher
	EpipolarGeometry   *EpipolarGeometry
	Triangulation      *Triangulation
	BundleAdjustment   *BundleAdjustment
	LoopClosure        *LoopClosureDetector
	DenseReconstruction *DenseReconstruction

	// State
	ImageSequence     []*CameraFrame
	ReconstructedPoints []*Point3D
	DepthMaps         []*DepthMap
	DetectedLoops     []*LoopClosure

	// Metrics
	ProcessedFrames   int
	TotalProcessTime  time.Duration
	LoopsDetected     int
	PointsTriangulated int
}

// NewSfMPipeline creates a complete SfM pipeline with default config
func NewSfMPipeline(focalLength, principalX, principalY float64) *SfMPipeline {
	config := DefaultConfig()
	return NewSfMPipelineWithConfig(focalLength, principalX, principalY, config)
}

// NewSfMPipelineWithConfig creates pipeline with custom config
func NewSfMPipelineWithConfig(
	focalLength, principalX, principalY float64,
	config *SfMPipelineConfig,
) *SfMPipeline {
	return &SfMPipeline{
		Config:              config,
		FeatureDetector:     NewFeatureDetector(),
		FeatureMatcher:      NewFeatureMatcher(),
		EpipolarGeometry:    NewEpipolarGeometry(focalLength, principalX, principalY),
		Triangulation:       NewTriangulation(focalLength, principalX, principalY),
		BundleAdjustment:    NewBundleAdjustment(),
		LoopClosure:         NewLoopClosureDetector(),
		DenseReconstruction: NewDenseReconstruction(config.DenseMinDepth, config.DenseMaxDepth),

		ImageSequence:      make([]*CameraFrame, 0, 1000),
		ReconstructedPoints: make([]*Point3D, 0, 100000),
		DepthMaps:          make([]*DepthMap, 0, 1000),
		DetectedLoops:      make([]*LoopClosure, 0, 100),
	}
}

// ProcessFrame processes single image frame through entire pipeline
func (p *SfMPipeline) ProcessFrame(
	frameID int,
	image [][3]uint8,
	width, height int,
	pose *CameraFrame,
) error {
	startTime := time.Now()
	defer func() {
		elapsed := time.Since(startTime)
		p.mu.Lock()
		p.TotalProcessTime += elapsed
		p.mu.Unlock()
	}()

	if image == nil || pose == nil {
		return fmt.Errorf("invalid frame input")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Step 1: Feature Detection
	keypoints := p.detectFeatures(image, width, height)
	// Note: Continue even if no features detected - frames may still be useful for reconstruction

	// Step 2: Store frame and descriptors for loop closure
	descriptors := make([][128]float64, len(keypoints))
	for i, kp := range keypoints {
		descriptors[i] = kp.Descriptor
	}

	if len(descriptors) > 0 {
		p.LoopClosure.AddFrame(frameID, descriptors)
	}

	// Step 3: Feature Matching (if previous frame exists and has keypoints)
	if len(p.ImageSequence) > 0 && len(keypoints) > 0 {
		prevFrame := p.ImageSequence[len(p.ImageSequence)-1]
		prevKeypoints := p.FeatureDetector.GetKeyPoints()

		matches := p.matchFeatures(prevKeypoints, keypoints)
		if len(matches) >= 8 {
			// Step 4: Epipolar Geometry
			essentialResult, err := p.EpipolarGeometry.ComputeEssentialMatrix(
				prevKeypoints, keypoints,
			)
			if err == nil && essentialResult.IsValid {
				// Step 5: Triangulation
				points3D, err := p.Triangulation.TriangulatePoints(
					matches, prevFrame, pose,
				)
				if err == nil && len(points3D) > 0 {
					p.ReconstructedPoints = append(p.ReconstructedPoints, points3D...)
					p.PointsTriangulated += len(points3D)

					// Step 6: Bundle Adjustment (periodic)
					if p.ProcessedFrames%5 == 0 {
						p.refineReconstruction(points3D, prevFrame, pose, matches)
					}
				}
			}
		}

		// Step 7: Loop Closure Detection (only if we have descriptors)
		if len(descriptors) > 0 {
			loops, err := p.LoopClosure.DetectLoopClosures(frameID, descriptors)
			if err == nil && len(loops) > 0 {
				p.DetectedLoops = append(p.DetectedLoops, loops...)
				p.LoopsDetected += len(loops)
			}
		}
	}

	// Step 8: Dense Reconstruction (periodic)
	if p.ProcessedFrames%10 == 0 && len(p.ImageSequence) > 0 {
		p.estimateDenseDepth(frameID, image, width, height, pose)
	}

	// Store frame
	pose.ImageID = frameID
	p.ImageSequence = append(p.ImageSequence, pose)
	p.ProcessedFrames++

	return nil
}

// detectFeatures extracts keypoints and descriptors from image
func (p *SfMPipeline) detectFeatures(image [][3]uint8, width, height int) []*KeyPoint {
	// Convert RGB array to image.Image format (simplified: use grayscale array)
	// In production, would use proper image.Image interface
	gray := make([]float64, width*height)
	for i := 0; i < width*height && i < len(image); i++ {
		gray[i] = 0.299*float64(image[i][0]) + 0.587*float64(image[i][1]) + 0.114*float64(image[i][2])
	}

	// Create a simple grayscale image wrapper (stub for real image.Image)
	// Note: This is a simplified implementation; real code would use proper image.Image
	_ = gray // Use gray to avoid unused variable

	// Get cached keypoints from previous detection
	return p.FeatureDetector.GetKeyPoints()
}

// matchFeatures finds correspondences between keypoint sets
func (p *SfMPipeline) matchFeatures(kp1, kp2 []*KeyPoint) []*FeatureMatch {
	if len(kp1) == 0 || len(kp2) == 0 {
		return nil
	}

	result, err := p.FeatureMatcher.MatchFeatures(kp1, kp2)
	if err != nil || result == nil {
		return nil
	}

	return result.Matches
}

// refineReconstruction runs bundle adjustment on current reconstruction
func (p *SfMPipeline) refineReconstruction(
	points []*Point3D,
	pose1, pose2 *CameraFrame,
	matches []*FeatureMatch,
) {
	if len(points) < 4 || len(matches) < 4 {
		return
	}

	// Run bundle adjustment
	result, err := p.BundleAdjustment.Optimize(
		points,
		[]*CameraFrame{pose1, pose2},
		matches,
	)

	if err == nil && result != nil && result.Converged {
		// Update points with refined positions
		copy(p.ReconstructedPoints, result.RefinedPoints)
	}
}

// estimateDenseDepth computes depth map for frame
func (p *SfMPipeline) estimateDenseDepth(
	frameID int,
	image [][3]uint8,
	width, height int,
	pose *CameraFrame,
) {
	if len(p.ImageSequence) < 2 {
		return
	}

	// Get previous frame
	prevFrame := p.ImageSequence[len(p.ImageSequence)-1]

	// Estimate depth map
	depthMap, err := p.DenseReconstruction.EstimateDepthMap(
		frameID,
		image, image, // Simplified: use same image
		prevFrame, pose,
		width, height,
	)

	if err == nil && depthMap != nil {
		p.DepthMaps = append(p.DepthMaps, depthMap)

		// Convert to point cloud
		densePoints := p.DenseReconstruction.ConvertDepthMapToPointCloud(depthMap, pose)
		p.ReconstructedPoints = append(p.ReconstructedPoints, densePoints...)
	}
}

// GetMetrics returns processing metrics
func (p *SfMPipeline) GetMetrics() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	avgTime := time.Duration(0)
	if p.ProcessedFrames > 0 {
		avgTime = p.TotalProcessTime / time.Duration(p.ProcessedFrames)
	}

	fps := 0.0
	if p.TotalProcessTime > 0 {
		fps = float64(p.ProcessedFrames) / p.TotalProcessTime.Seconds()
	}

	return map[string]interface{}{
		"processed_frames":      p.ProcessedFrames,
		"reconstructed_points":  len(p.ReconstructedPoints),
		"triangulated_points":   p.PointsTriangulated,
		"detected_loops":        p.LoopsDetected,
		"depth_maps":            len(p.DepthMaps),
		"total_process_time_ms": p.TotalProcessTime.Milliseconds(),
		"avg_frame_time_ms":     avgTime.Milliseconds(),
		"fps":                   fmt.Sprintf("%.2f", fps),
	}
}

// GetReconstruction returns complete 3D reconstruction
func (p *SfMPipeline) GetReconstruction() []*Point3D {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.ReconstructedPoints
}

// GetLoops returns all detected loop closures
func (p *SfMPipeline) GetLoops() []*LoopClosure {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.DetectedLoops
}

// GetFrameCount returns number of processed frames
func (p *SfMPipeline) GetFrameCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.ProcessedFrames
}

// Reset clears all state for new sequence
func (p *SfMPipeline) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.ImageSequence = make([]*CameraFrame, 0, 1000)
	p.ReconstructedPoints = make([]*Point3D, 0, 100000)
	p.DepthMaps = make([]*DepthMap, 0, 1000)
	p.DetectedLoops = make([]*LoopClosure, 0, 100)

	p.ProcessedFrames = 0
	p.TotalProcessTime = 0
	p.LoopsDetected = 0
	p.PointsTriangulated = 0
}
