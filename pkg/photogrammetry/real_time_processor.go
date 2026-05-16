package photogrammetry

import (
	"fmt"
	"sync"
	"time"
)

// ProcessingMetrics stores real-time performance metrics
type ProcessingMetrics struct {
	FrameID              int
	ProcessingTimeMs     float64
	FeatureDetectionMs   float64
	FeatureMatchingMs    float64
	EpipolarGeometryMs   float64
	TriangulationMs      float64
	BundleAdjustmentMs   float64
	DenseReconstructionMs float64
	IsKeyframe           bool
	DropFrame            bool
	Timestamp            time.Time
}

// RealTimeProcessor optimizes SfM pipeline for real-time performance
// Targets 10 Hz (100ms per frame) processing
type RealTimeProcessor struct {
	mu sync.RWMutex

	// Target parameters
	TargetFPS          int           // 10 Hz
	TargetFrameTimeMs  float64       // 100 ms
	MaxDropRatePercent float64       // 10% max frames can be dropped

	// Components
	Pipeline         *SfMPipeline
	KeyframeSelector *KeyframeSelector

	// Processing strategy
	SkipDenseEveryN  int     // Process dense reconstruction every N frames
	SkipBAEveryN     int     // Process bundle adjustment every N frames
	DynamicFrameSkip bool    // Enable frame dropping if slow
	CurrentDropRate  float64 // Current percentage of frames dropped

	// Statistics
	ProcessedFrames  int
	SkippedFrames    int
	AverageTimeMs    float64
	PeakTimeMs       float64
	MetricsHistory   []*ProcessingMetrics
}

// NewRealTimeProcessor creates a processor for real-time SfM pipeline
func NewRealTimeProcessor(pipeline *SfMPipeline) *RealTimeProcessor {
	return &RealTimeProcessor{
		TargetFPS:         10,
		TargetFrameTimeMs: 100.0,
		MaxDropRatePercent: 10.0,
		Pipeline:          pipeline,
		KeyframeSelector:  NewKeyframeSelector(),
		SkipDenseEveryN:   10,
		SkipBAEveryN:      5,
		DynamicFrameSkip:  true,
		MetricsHistory:    make([]*ProcessingMetrics, 0, 1000),
	}
}

// ProcessFrameRealTime processes frame with real-time optimization
func (rtp *RealTimeProcessor) ProcessFrameRealTime(
	frameID int,
	image [][3]uint8,
	width, height int,
	pose *CameraFrame,
) (*ProcessingMetrics, error) {
	startTime := time.Now()

	if image == nil || pose == nil {
		return nil, fmt.Errorf("invalid frame input")
	}

	rtp.mu.Lock()
	defer rtp.mu.Unlock()

	metrics := &ProcessingMetrics{
		FrameID:   frameID,
		Timestamp: startTime,
	}

	// Check if frame should be skipped
	if rtp.shouldSkipFrame(frameID) {
		metrics.DropFrame = true
		rtp.SkippedFrames++
		return metrics, nil
	}

	// Step 1: Feature Detection
	t1 := time.Now()
	keypoints := rtp.Pipeline.detectFeatures(image, width, height)
	metrics.FeatureDetectionMs = float64(time.Since(t1).Milliseconds())

	// Step 2: Feature Matching (if enough keypoints)
	if len(keypoints) > rtp.KeyframeSelector.MinKeypointThreshold {
		t2 := time.Now()
		if len(rtp.Pipeline.ImageSequence) > 0 {
			prevKeypoints := rtp.Pipeline.FeatureDetector.GetKeyPoints()
			matchResult, _ := rtp.Pipeline.FeatureMatcher.MatchFeatures(prevKeypoints, keypoints)
			if matchResult != nil {
				metrics.FeatureMatchingMs = float64(time.Since(t2).Milliseconds())
			}
		}
	}

	// Step 3: Keyframe Selection
	matchResult := &FeatureMatchResult{
		MatchCount: len(keypoints),
		SuccessRate: 0.7, // Simplified
	}
	keyframeInfo := rtp.KeyframeSelector.EvaluateFrame(
		frameID, len(keypoints), matchResult, 0.05, 0.05,
	)

	metrics.IsKeyframe = keyframeInfo.IsKeyframe

	// Step 4: Skip expensive operations for non-keyframes
	if !keyframeInfo.IsKeyframe && rtp.DynamicFrameSkip {
		// Fast path for non-keyframes
		rtp.ProcessedFrames++
		return rtp.recordMetrics(metrics), nil
	}

	// Step 5: Full processing for keyframes
	t3 := time.Now()
	err := rtp.Pipeline.ProcessFrame(frameID, image, width, height, pose)
	if err != nil {
		// Non-critical error - log but continue
	}
	totalTime := float64(time.Since(t3).Milliseconds())
	metrics.ProcessingTimeMs = totalTime

	// Step 6: Adaptive threshold adjustment
	rtp.KeyframeSelector.AdaptiveThreshold(totalTime, rtp.TargetFrameTimeMs)

	// Step 7: Check if we're exceeding time budget
	if totalTime > rtp.TargetFrameTimeMs*1.5 {
		// Too slow - will skip more frames next time
		rtp.CurrentDropRate = rtp.CurrentDropRate*0.9 + 0.1*10.0 // Increase drop rate
	} else if totalTime < rtp.TargetFrameTimeMs*0.7 {
		// Fast - can process more frames
		rtp.CurrentDropRate = rtp.CurrentDropRate * 0.95
	}

	rtp.ProcessedFrames++

	return rtp.recordMetrics(metrics), nil
}

// shouldSkipFrame determines if frame should be skipped for speed
func (rtp *RealTimeProcessor) shouldSkipFrame(frameID int) bool {
	if !rtp.DynamicFrameSkip {
		return false
	}

	// Skip if we've exceeded drop rate tolerance
	if rtp.SkippedFrames+1 > int(float64(rtp.ProcessedFrames+rtp.SkippedFrames) * rtp.CurrentDropRate / 100.0) {
		return false
	}

	// Skip non-critical frames based on drop rate
	return (frameID % int(100.0/rtp.CurrentDropRate)) != 0
}

// recordMetrics records and updates statistics
func (rtp *RealTimeProcessor) recordMetrics(metrics *ProcessingMetrics) *ProcessingMetrics {
	rtp.MetricsHistory = append(rtp.MetricsHistory, metrics)

	// Update statistics
	if metrics.ProcessingTimeMs > 0 {
		if rtp.PeakTimeMs == 0 || metrics.ProcessingTimeMs > rtp.PeakTimeMs {
			rtp.PeakTimeMs = metrics.ProcessingTimeMs
		}

		// Update average
		totalTime := 0.0
		for _, m := range rtp.MetricsHistory {
			if m.ProcessingTimeMs > 0 {
				totalTime += m.ProcessingTimeMs
			}
		}
		count := 0
		for _, m := range rtp.MetricsHistory {
			if m.ProcessingTimeMs > 0 {
				count++
			}
		}
		if count > 0 {
			rtp.AverageTimeMs = totalTime / float64(count)
		}
	}

	return metrics
}

// GetThroughput returns frames per second
func (rtp *RealTimeProcessor) GetThroughput() float64 {
	rtp.mu.RLock()
	defer rtp.mu.RUnlock()

	if len(rtp.MetricsHistory) == 0 {
		return 0.0
	}

	// Calculate FPS from metrics timestamps
	if len(rtp.MetricsHistory) > 1 {
		first := rtp.MetricsHistory[0].Timestamp
		last := rtp.MetricsHistory[len(rtp.MetricsHistory)-1].Timestamp
		duration := last.Sub(first)
		if duration > 0 {
			return float64(len(rtp.MetricsHistory)) / duration.Seconds()
		}
	}

	return float64(rtp.ProcessedFrames) / (rtp.AverageTimeMs / 1000.0)
}

// GetMetricsReport returns performance report
func (rtp *RealTimeProcessor) GetMetricsReport() map[string]interface{} {
	rtp.mu.RLock()
	defer rtp.mu.RUnlock()

	report := map[string]interface{}{
		"target_fps":          rtp.TargetFPS,
		"actual_fps":          fmt.Sprintf("%.2f", rtp.GetThroughput()),
		"average_frame_time_ms": fmt.Sprintf("%.2f", rtp.AverageTimeMs),
		"peak_frame_time_ms":  fmt.Sprintf("%.2f", rtp.PeakTimeMs),
		"processed_frames":    rtp.ProcessedFrames,
		"skipped_frames":      rtp.SkippedFrames,
		"current_drop_rate":   fmt.Sprintf("%.2f%%", rtp.CurrentDropRate),
		"keyframe_rate":       fmt.Sprintf("%.2f%%", rtp.KeyframeSelector.GetSelectionRate()*100),
	}

	return report
}

// GetMetricsHistory returns raw metrics for analysis
func (rtp *RealTimeProcessor) GetMetricsHistory() []*ProcessingMetrics {
	rtp.mu.RLock()
	defer rtp.mu.RUnlock()
	return rtp.MetricsHistory
}

// ResetMetrics clears metrics history
func (rtp *RealTimeProcessor) ResetMetrics() {
	rtp.mu.Lock()
	defer rtp.mu.Unlock()

	rtp.ProcessedFrames = 0
	rtp.SkippedFrames = 0
	rtp.AverageTimeMs = 0.0
	rtp.PeakTimeMs = 0.0
	rtp.MetricsHistory = make([]*ProcessingMetrics, 0, 1000)
}

// GetKeyframeSelector returns the keyframe selector
func (rtp *RealTimeProcessor) GetKeyframeSelector() *KeyframeSelector {
	rtp.mu.RLock()
	defer rtp.mu.RUnlock()
	return rtp.KeyframeSelector
}

// GetPipeline returns the SfM pipeline
func (rtp *RealTimeProcessor) GetPipeline() *SfMPipeline {
	rtp.mu.RLock()
	defer rtp.mu.RUnlock()
	return rtp.Pipeline
}

// SetDynamicFrameSkip enables/disables dynamic frame skipping
func (rtp *RealTimeProcessor) SetDynamicFrameSkip(enabled bool) {
	rtp.mu.Lock()
	defer rtp.mu.Unlock()
	rtp.DynamicFrameSkip = enabled
}

// SetTargetFPS changes target FPS
func (rtp *RealTimeProcessor) SetTargetFPS(fps int) error {
	if fps <= 0 || fps > 60 {
		return fmt.Errorf("invalid FPS: %d (must be 1-60)", fps)
	}

	rtp.mu.Lock()
	defer rtp.mu.Unlock()

	rtp.TargetFPS = fps
	rtp.TargetFrameTimeMs = 1000.0 / float64(fps)

	return nil
}
