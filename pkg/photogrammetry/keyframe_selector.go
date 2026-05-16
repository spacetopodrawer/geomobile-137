package photogrammetry

import (
	"fmt"
	"sync"
)

// KeyframeInfo stores metadata about a potential keyframe
type KeyframeInfo struct {
	FrameID         int
	KeypointCount   int
	MatchQuality    float64    // Match success rate [0, 1]
	TranslationMag  float64    // Camera motion magnitude
	RotationMag     float64    // Camera rotation magnitude
	TrackingScore   float64    // Composite tracking quality [0, 1]
	IsKeyframe      bool
	Timestamp       int64
}

// KeyframeSelector intelligently selects keyframes to optimize processing
// Balances reconstruction quality with computational efficiency
type KeyframeSelector struct {
	mu sync.RWMutex

	// Selection criteria
	MinKeypointThreshold   int     // Minimum keypoints for keyframe
	MinTranslationThreshold float64 // Minimum camera motion
	MinTrackingScore       float64 // Minimum tracking quality
	MinFrameGap            int     // Minimum frames between keyframes

	// State
	FrameHistory       []*KeyframeInfo
	SelectedKeyframes  []*KeyframeInfo
	KeyframeCount      int
	TotalFramesProcessed int
}

// NewKeyframeSelector creates a new keyframe selection engine
func NewKeyframeSelector() *KeyframeSelector {
	return &KeyframeSelector{
		MinKeypointThreshold:    50,    // At least 50 keypoints
		MinTranslationThreshold: 0.01,  // At least 1cm camera motion
		MinTrackingScore:        0.6,   // At least 60% quality
		MinFrameGap:             3,     // At least 3 frames apart
		FrameHistory:            make([]*KeyframeInfo, 0, 1000),
		SelectedKeyframes:       make([]*KeyframeInfo, 0, 100),
	}
}

// EvaluateFrame evaluates if frame should be a keyframe
func (ks *KeyframeSelector) EvaluateFrame(
	frameID int,
	keypointCount int,
	matchResult *FeatureMatchResult,
	translationMag, rotationMag float64,
) *KeyframeInfo {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	info := &KeyframeInfo{
		FrameID:        frameID,
		KeypointCount:  keypointCount,
		TranslationMag: translationMag,
		RotationMag:    rotationMag,
		Timestamp:      int64(frameID),
	}

	// Compute match quality
	if matchResult != nil {
		info.MatchQuality = matchResult.SuccessRate
	}

	// Compute composite tracking score
	info.TrackingScore = ks.computeTrackingScore(info)

	// Decide if this should be a keyframe
	info.IsKeyframe = ks.shouldBeKeyframe(info)

	// Track frame
	ks.FrameHistory = append(ks.FrameHistory, info)
	ks.TotalFramesProcessed++

	if info.IsKeyframe {
		ks.SelectedKeyframes = append(ks.SelectedKeyframes, info)
		ks.KeyframeCount++
	}

	return info
}

// shouldBeKeyframe determines if frame meets keyframe criteria
func (ks *KeyframeSelector) shouldBeKeyframe(info *KeyframeInfo) bool {
	// Check minimum keypoint threshold
	if info.KeypointCount < ks.MinKeypointThreshold {
		return false
	}

	// Check minimum tracking quality
	if info.TrackingScore < ks.MinTrackingScore {
		return false
	}

	// Check if enough motion occurred
	hasMotion := info.TranslationMag >= ks.MinTranslationThreshold ||
		info.RotationMag >= 0.05 // ~3 degrees

	if !hasMotion {
		return false
	}

	// Check minimum frame gap from last keyframe
	if len(ks.SelectedKeyframes) > 0 {
		lastKeyframe := ks.SelectedKeyframes[len(ks.SelectedKeyframes)-1]
		if info.FrameID-lastKeyframe.FrameID < ks.MinFrameGap {
			return false
		}
	}

	return true
}

// computeTrackingScore computes composite quality metric
func (ks *KeyframeSelector) computeTrackingScore(info *KeyframeInfo) float64 {
	// Weight: 50% match quality, 30% keypoints, 20% motion
	keypointScore := float64(info.KeypointCount) / 200.0 // Normalize to ~200 expected
	if keypointScore > 1.0 {
		keypointScore = 1.0
	}

	motionScore := info.TranslationMag / 0.1 // Normalize to ~10cm
	if motionScore > 1.0 {
		motionScore = 1.0
	}

	score := 0.5*info.MatchQuality + 0.3*keypointScore + 0.2*motionScore
	return score
}

// GetSelectedKeyframes returns all selected keyframes
func (ks *KeyframeSelector) GetSelectedKeyframes() []*KeyframeInfo {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	return ks.SelectedKeyframes
}

// GetKeyframeCount returns number of selected keyframes
func (ks *KeyframeSelector) GetKeyframeCount() int {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	return ks.KeyframeCount
}

// GetSelectionRate returns percentage of frames selected as keyframes
func (ks *KeyframeSelector) GetSelectionRate() float64 {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	if ks.TotalFramesProcessed == 0 {
		return 0.0
	}

	return float64(ks.KeyframeCount) / float64(ks.TotalFramesProcessed)
}

// Reset clears selection history
func (ks *KeyframeSelector) Reset() {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	ks.FrameHistory = make([]*KeyframeInfo, 0, 1000)
	ks.SelectedKeyframes = make([]*KeyframeInfo, 0, 100)
	ks.KeyframeCount = 0
	ks.TotalFramesProcessed = 0
}

// GetFrameHistory returns all evaluated frames
func (ks *KeyframeSelector) GetFrameHistory() []*KeyframeInfo {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	return ks.FrameHistory
}

// ForceKeyframe forces a frame to be a keyframe (for critical frames)
func (ks *KeyframeSelector) ForceKeyframe(frameID int) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	// Find frame in history
	for _, info := range ks.FrameHistory {
		if info.FrameID == frameID && !info.IsKeyframe {
			info.IsKeyframe = true
			ks.SelectedKeyframes = append(ks.SelectedKeyframes, info)
			ks.KeyframeCount++
			return nil
		}
	}

	return fmt.Errorf("frame %d not found in history", frameID)
}

// AdaptiveThreshold adjusts selection thresholds based on processing performance
func (ks *KeyframeSelector) AdaptiveThreshold(processingTimeMs float64, targetTimeMs float64) {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	ratio := processingTimeMs / targetTimeMs

	if ratio > 1.5 {
		// Too slow - increase thresholds to select fewer keyframes
		ks.MinKeypointThreshold = (ks.MinKeypointThreshold * 11) / 10 // +10%
		ks.MinTrackingScore = (ks.MinTrackingScore * 101) / 100      // +1%
	} else if ratio < 0.7 {
		// Fast enough - lower thresholds to select more keyframes
		ks.MinKeypointThreshold = (ks.MinKeypointThreshold * 9) / 10  // -10%
		ks.MinTrackingScore = (ks.MinTrackingScore * 99) / 100        // -1%
	}

	// Clamp values to reasonable ranges
	if ks.MinKeypointThreshold < 20 {
		ks.MinKeypointThreshold = 20
	}
	if ks.MinKeypointThreshold > 200 {
		ks.MinKeypointThreshold = 200
	}
	if ks.MinTrackingScore < 0.4 {
		ks.MinTrackingScore = 0.4
	}
	if ks.MinTrackingScore > 0.9 {
		ks.MinTrackingScore = 0.9
	}
}
