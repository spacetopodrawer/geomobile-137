package photogrammetry

import (
	"fmt"
	"sync"
	"time"
)

// ArcadeFrameInfo stores metadata for arcade game frames
type ArcadeFrameInfo struct {
	FrameID          int
	Timestamp        time.Time
	SceneType        string    // "2D", "3D", "UI", "Unknown"
	TextureRegions   []Region  // Detected texture regions
	FeatureDensity   float64   // Features per pixel [0, 1]
	ReconstError     float64   // Reconstruction error [0, 1]
	ReconstructOK    bool      // Reconstruction successful
	ProcessingTimeMs float64
}

// Region represents a detected region in arcade frame
type Region struct {
	X, Y, Width, Height int
	Type                string    // "sprite", "background", "ui"
	Confidence          float64
	FeatureCount        int
}

// ArcadeProcessor handles arcade game-specific image processing
// Optimized for 2D sprites, 3D geometry, and arcade-specific patterns
type ArcadeProcessor struct {
	mu sync.RWMutex

	// Configuration
	TargetWidth       int     // Expected arcade frame width
	TargetHeight      int     // Expected arcade frame height
	TextureThreshold  float64 // Min feature density for texture [0, 1]
	SceneConfidence   float64 // Min confidence for scene classification

	// Pipeline integration
	RealTimeProcessor *RealTimeProcessor
	Pipeline          *SfMPipeline

	// State
	ProcessedFrames    int
	SuccessfulFrames   int
	FailedFrames       int
	AverageError       float64
	FrameHistory       []*ArcadeFrameInfo
	ReconstructedScene *ReconstructedArcadeScene
}

// ReconstructedArcadeScene holds complete reconstruction of arcade game
type ReconstructedArcadeScene struct {
	SparsePoints    []*Point3D
	DensePoints     []*Point3D
	CameraTrajectory []*CameraFrame
	LoopClosures    []*LoopClosure
	Timestamp       time.Time
	TotalFrames     int
	KeyframeCount   int
}

// NewArcadeProcessor creates processor for arcade game images
func NewArcadeProcessor(width, height int, pipeline *SfMPipeline, rtp *RealTimeProcessor) *ArcadeProcessor {
	return &ArcadeProcessor{
		TargetWidth:       width,
		TargetHeight:      height,
		TextureThreshold:  0.3,
		SceneConfidence:   0.7,
		RealTimeProcessor: rtp,
		Pipeline:          pipeline,
		FrameHistory:      make([]*ArcadeFrameInfo, 0, 1000),
	}
}

// ProcessArcadeFrame processes single arcade game frame
func (ap *ArcadeProcessor) ProcessArcadeFrame(
	frameID int,
	image [][3]uint8,
	estimatedPose *CameraFrame,
) (*ArcadeFrameInfo, error) {
	startTime := time.Now()

	if image == nil || estimatedPose == nil {
		return nil, fmt.Errorf("invalid frame input")
	}

	ap.mu.Lock()
	defer ap.mu.Unlock()

	info := &ArcadeFrameInfo{
		FrameID:   frameID,
		Timestamp: startTime,
		SceneType: "Unknown",
	}

	// Step 1: Detect scene type (2D sprites vs 3D geometry)
	sceneType := ap.classifyArcadeScene(image)
	info.SceneType = sceneType

	// Step 2: Detect texture regions (for sprites or geometry)
	regions := ap.detectTextureRegions(image)
	info.TextureRegions = regions

	// Step 3: Compute feature density
	keypoints := ap.Pipeline.FeatureDetector.GetKeyPoints()
	info.FeatureDensity = float64(len(keypoints)) / float64(len(image))

	// Step 4: Process through real-time pipeline
	metrics, err := ap.RealTimeProcessor.ProcessFrameRealTime(
		frameID, image, ap.TargetWidth, ap.TargetHeight, estimatedPose,
	)

	if err == nil && metrics != nil {
		info.ProcessingTimeMs = metrics.ProcessingTimeMs
		info.ReconstructOK = !metrics.DropFrame
	}

	// Step 5: Compute reconstruction error
	if len(ap.Pipeline.ReconstructedPoints) > 0 {
		info.ReconstError = ap.computeReconsError(image, estimatedPose)
		info.ReconstructOK = info.ReconstError < 0.3
	}

	// Update statistics
	ap.FrameHistory = append(ap.FrameHistory, info)
	ap.ProcessedFrames++
	if info.ReconstructOK {
		ap.SuccessfulFrames++
	} else {
		ap.FailedFrames++
	}

	// Update average error
	totalError := 0.0
	for _, frame := range ap.FrameHistory {
		totalError += frame.ReconstError
	}
	ap.AverageError = totalError / float64(len(ap.FrameHistory))

	return info, nil
}

// classifyArcadeScene detects if frame contains 2D sprites or 3D geometry
func (ap *ArcadeProcessor) classifyArcadeScene(image [][3]uint8) string {
	if len(image) == 0 {
		return "Unknown"
	}

	// Analyze color histogram
	colorVariance := ap.computeColorVariance(image)

	// Arcade 2D sprites typically have:
	// - High color variance (bright colors)
	// - Sharp edges (high gradients)
	if colorVariance > 0.6 {
		return "2D"
	}

	// Arcade 3D games have:
	// - Smoother color transitions
	// - More gradient complexity
	if colorVariance > 0.4 {
		return "3D"
	}

	return "UI"
}

// computeColorVariance estimates color distribution variance
func (ap *ArcadeProcessor) computeColorVariance(image [][3]uint8) float64 {
	if len(image) == 0 {
		return 0.0
	}

	// Sample pixels
	var rSum, gSum, bSum float64
	sampleSize := len(image)
	if sampleSize > 1000 {
		sampleSize = 1000 // Limit sampling for speed
	}

	for i := 0; i < sampleSize && i < len(image); i++ {
		rSum += float64(image[i][0])
		gSum += float64(image[i][1])
		bSum += float64(image[i][2])
	}

	rMean := rSum / float64(sampleSize)
	gMean := gSum / float64(sampleSize)
	bMean := bSum / float64(sampleSize)

	// Compute variance
	var rVar, gVar, bVar float64
	for i := 0; i < sampleSize && i < len(image); i++ {
		rDiff := float64(image[i][0]) - rMean
		gDiff := float64(image[i][1]) - gMean
		bDiff := float64(image[i][2]) - bMean
		rVar += rDiff * rDiff
		gVar += gDiff * gDiff
		bVar += bDiff * bDiff
	}

	totalVar := (rVar + gVar + bVar) / float64(sampleSize) / (256.0 * 256.0)
	return totalVar
}

// detectTextureRegions finds regions with consistent texture patterns
func (ap *ArcadeProcessor) detectTextureRegions(image [][3]uint8) []Region {
	regions := make([]Region, 0, 10)

	if len(image) == 0 || ap.TargetWidth == 0 || ap.TargetHeight == 0 {
		return regions
	}

	// Simplified: divide image into 4x4 grid
	regionWidth := ap.TargetWidth / 4
	regionHeight := ap.TargetHeight / 4

	for ry := 0; ry < 4; ry++ {
		for rx := 0; rx < 4; rx++ {
			x := rx * regionWidth
			y := ry * regionHeight

			// Count features in region
			featureCount := 0
			for yy := y; yy < y+regionHeight && yy < ap.TargetHeight; yy++ {
				for xx := x; xx < x+regionWidth && xx < ap.TargetWidth; xx++ {
					idx := yy*ap.TargetWidth + xx
					if idx < len(image) {
						// Simplified: count non-gray pixels as features
						r, g, b := image[idx][0], image[idx][1], image[idx][2]
						if r != g || g != b {
							featureCount++
						}
					}
				}
			}

			confidence := float64(featureCount) / float64(regionWidth*regionHeight)
			if confidence > ap.TextureThreshold {
				region := Region{
					X:             x,
					Y:             y,
					Width:         regionWidth,
					Height:        regionHeight,
					Type:          "sprite",
					Confidence:    confidence,
					FeatureCount:  featureCount,
				}
				regions = append(regions, region)
			}
		}
	}

	return regions
}

// computeReconsError estimates reconstruction quality
func (ap *ArcadeProcessor) computeReconsError(image [][3]uint8, pose *CameraFrame) float64 {
	if len(ap.Pipeline.ReconstructedPoints) == 0 {
		return 1.0 // Perfect error = 1.0 (worst)
	}

	// Simplified: use point cloud density as quality measure
	pointCount := len(ap.Pipeline.ReconstructedPoints)
	pixelCount := len(image)

	if pixelCount == 0 {
		return 1.0
	}

	density := float64(pointCount) / float64(pixelCount)

	// Expected density for good reconstruction: ~0.1 (10% of pixels)
	// Error: |density - 0.1| normalized
	expectedDensity := 0.1
	error := 1.0 - (1.0 / (1.0 + (density - expectedDensity) * (density - expectedDensity)))

	return error
}

// GetSuccessRate returns percentage of successful frames
func (ap *ArcadeProcessor) GetSuccessRate() float64 {
	ap.mu.RLock()
	defer ap.mu.RUnlock()

	if ap.ProcessedFrames == 0 {
		return 0.0
	}

	return float64(ap.SuccessfulFrames) / float64(ap.ProcessedFrames)
}

// GetFrameInfo returns info for specific frame
func (ap *ArcadeProcessor) GetFrameInfo(frameID int) *ArcadeFrameInfo {
	ap.mu.RLock()
	defer ap.mu.RUnlock()

	for _, info := range ap.FrameHistory {
		if info.FrameID == frameID {
			return info
		}
	}

	return nil
}

// GetSceneStatistics returns statistics about processed scene
func (ap *ArcadeProcessor) GetSceneStatistics() map[string]interface{} {
	ap.mu.RLock()
	defer ap.mu.RUnlock()

	scene2D := 0
	scene3D := 0

	for _, info := range ap.FrameHistory {
		if info.SceneType == "2D" {
			scene2D++
		} else if info.SceneType == "3D" {
			scene3D++
		}
	}

	return map[string]interface{}{
		"total_frames":      ap.ProcessedFrames,
		"successful_frames": ap.SuccessfulFrames,
		"failed_frames":     ap.FailedFrames,
		"success_rate":      fmt.Sprintf("%.2f%%", ap.GetSuccessRate()*100),
		"average_error":     fmt.Sprintf("%.3f", ap.AverageError),
		"scene_2d_count":    scene2D,
		"scene_3d_count":    scene3D,
	}
}

// FinalizeReconstruction creates final 3D model from all frames
func (ap *ArcadeProcessor) FinalizeReconstruction() *ReconstructedArcadeScene {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	scene := &ReconstructedArcadeScene{
		SparsePoints:     ap.Pipeline.GetReconstruction(),
		CameraTrajectory: ap.Pipeline.ImageSequence,
		LoopClosures:     ap.Pipeline.GetLoops(),
		Timestamp:        time.Now(),
		TotalFrames:      ap.ProcessedFrames,
		KeyframeCount:    ap.RealTimeProcessor.GetKeyframeSelector().GetKeyframeCount(),
	}

	// Collect dense points from all depth maps
	for _, depthMap := range ap.Pipeline.DepthMaps {
		if depthMap != nil && len(scene.CameraTrajectory) > 0 {
			// Use last pose for depth map projection
			lastPose := scene.CameraTrajectory[len(scene.CameraTrajectory)-1]
			densePoints := ap.Pipeline.DenseReconstruction.ConvertDepthMapToPointCloud(depthMap, lastPose)
			scene.DensePoints = append(scene.DensePoints, densePoints...)
		}
	}

	ap.ReconstructedScene = scene
	return scene
}

// GetReconstructedScene returns final 3D model
func (ap *ArcadeProcessor) GetReconstructedScene() *ReconstructedArcadeScene {
	ap.mu.RLock()
	defer ap.mu.RUnlock()
	return ap.ReconstructedScene
}

// Reset clears all state
func (ap *ArcadeProcessor) Reset() {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	ap.ProcessedFrames = 0
	ap.SuccessfulFrames = 0
	ap.FailedFrames = 0
	ap.AverageError = 0.0
	ap.FrameHistory = make([]*ArcadeFrameInfo, 0, 1000)
	ap.ReconstructedScene = nil
}
