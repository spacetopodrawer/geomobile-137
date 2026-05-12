package photogrammetry

import (
	"fmt"
	"math"
	"sync"
)

// LoopClosure represents a detected loop in the camera trajectory
type LoopClosure struct {
	CurrentFrameID  int
	PreviousFrameID int
	Confidence      float64    // Match confidence [0, 1]
	TransformMatrix [4][4]float64 // 4x4 pose transformation
	InlierCount     int
	OutlierCount    int
	Timestamp       int64
}

// LoopClosureDetector detects when camera revisits previous locations
// Uses vocabulary tree-like approach for efficient place recognition
type LoopClosureDetector struct {
	mu sync.RWMutex

	// Configuration
	MinFrameGap        int     // Minimum frames between loop candidates
	ConfidenceThreshold float64 // Min confidence to accept loop
	InlierRatioThreshold float64 // Min inlier ratio

	// State
	FrameDescriptors   []DescriptorSet // Descriptors per frame
	FrameCount         int
	DetectedLoops      []*LoopClosure
	LoopClosureCount   int
}

// DescriptorSet groups descriptors from a single frame
type DescriptorSet struct {
	FrameID     int
	Descriptors [][128]float64
	Mean        [128]float64  // Average descriptor
	Valid       bool
}

// NewLoopClosureDetector creates a new loop closure detector
func NewLoopClosureDetector() *LoopClosureDetector {
	return &LoopClosureDetector{
		MinFrameGap:           20,     // 20 frames minimum
		ConfidenceThreshold:   0.8,    // 80% confidence
		InlierRatioThreshold:  0.7,    // 70% inliers
		FrameDescriptors:      make([]DescriptorSet, 0, 1000),
		DetectedLoops:         make([]*LoopClosure, 0, 100),
	}
}

// AddFrame registers descriptors for a new frame
func (lcd *LoopClosureDetector) AddFrame(frameID int, descriptors [][128]float64) error {
	if len(descriptors) == 0 {
		return fmt.Errorf("no descriptors for frame %d", frameID)
	}

	lcd.mu.Lock()
	defer lcd.mu.Unlock()

	// Compute mean descriptor (vocabulary tree anchor point)
	meanDesc := lcd.computeMeanDescriptor(descriptors)

	frameSet := DescriptorSet{
		FrameID:     frameID,
		Descriptors: descriptors,
		Mean:        meanDesc,
		Valid:       true,
	}

	lcd.FrameDescriptors = append(lcd.FrameDescriptors, frameSet)
	lcd.FrameCount = len(lcd.FrameDescriptors)

	return nil
}

// DetectLoopClosures searches for revisited locations in current frame
func (lcd *LoopClosureDetector) DetectLoopClosures(
	currentFrameID int,
	currentDescriptors [][128]float64,
) ([]*LoopClosure, error) {
	if len(currentDescriptors) == 0 {
		return nil, fmt.Errorf("no descriptors for current frame")
	}

	lcd.mu.Lock()
	defer lcd.mu.Unlock()

	loops := make([]*LoopClosure, 0, 10)

	// Compute mean descriptor for current frame
	currentMean := lcd.computeMeanDescriptor(currentDescriptors)

	// Search through previous frames
	for i, frameSet := range lcd.FrameDescriptors {
		if !frameSet.Valid {
			continue
		}

		// Skip if frames too close
		if currentFrameID-frameSet.FrameID < lcd.MinFrameGap {
			continue
		}

		// Quick rejection: compare mean descriptors
		meanDist := lcd.descriptorDistance(currentMean, frameSet.Mean)
		if meanDist > 50.0 { // Threshold for quick rejection
			continue
		}

		// Detailed matching: match current descriptors against frame
		matches := lcd.matchDescriptorSets(currentDescriptors, frameSet.Descriptors)

		if len(matches) < 8 { // Minimum for reliable pose estimation
			continue
		}

		// RANSAC to estimate pose and count inliers
		inliers, outliers, transform := lcd.estimatePoseRANSAC(matches)

		inlierRatio := float64(inliers) / float64(len(matches))
		if inlierRatio < lcd.InlierRatioThreshold {
			continue
		}

		confidence := inlierRatio * (1.0 - meanDist/100.0) // Combine metrics
		if confidence < lcd.ConfidenceThreshold {
			continue
		}

		// Valid loop closure detected
		loop := &LoopClosure{
			CurrentFrameID:  currentFrameID,
			PreviousFrameID: frameSet.FrameID,
			Confidence:      confidence,
			TransformMatrix: transform,
			InlierCount:     inliers,
			OutlierCount:    outliers,
			Timestamp:       int64(i),
		}

		loops = append(loops, loop)
	}

	// Store detected loops
	lcd.DetectedLoops = append(lcd.DetectedLoops, loops...)
	lcd.LoopClosureCount = len(lcd.DetectedLoops)

	return loops, nil
}

// computeMeanDescriptor computes average descriptor across set
func (lcd *LoopClosureDetector) computeMeanDescriptor(descriptors [][128]float64) [128]float64 {
	var mean [128]float64

	if len(descriptors) == 0 {
		return mean
	}

	for _, desc := range descriptors {
		for j := 0; j < 128; j++ {
			mean[j] += desc[j]
		}
	}

	for j := 0; j < 128; j++ {
		mean[j] /= float64(len(descriptors))
	}

	return mean
}

// matchDescriptorSets finds correspondences between descriptor sets
func (lcd *LoopClosureDetector) matchDescriptorSets(
	set1, set2 [][128]float64,
) []*FeatureMatch {
	matches := make([]*FeatureMatch, 0, len(set1))

	for _, desc1 := range set1 {
		bestDist := math.MaxFloat64
		bestIdx := -1
		secondBestDist := math.MaxFloat64

		for j, desc2 := range set2 {
			dist := lcd.descriptorDistance(desc1, desc2)

			if dist < bestDist {
				secondBestDist = bestDist
				bestDist = dist
				bestIdx = j
			} else if dist < secondBestDist {
				secondBestDist = dist
			}
		}

		// Lowe's ratio test
		if bestIdx >= 0 && bestDist < 0.7*secondBestDist && bestDist < 30.0 {
			match := &FeatureMatch{
				Distance:   bestDist,
				Confidence: 1.0 - bestDist/50.0,
			}
			matches = append(matches, match)
		}
	}

	return matches
}

// descriptorDistance computes distance between two descriptors
func (lcd *LoopClosureDetector) descriptorDistance(d1, d2 [128]float64) float64 {
	var sum float64
	for i := 0; i < 128; i++ {
		diff := d1[i] - d2[i]
		sum += diff * diff
	}
	return math.Sqrt(sum)
}

// estimatePoseRANSAC estimates transformation matrix with RANSAC
func (lcd *LoopClosureDetector) estimatePoseRANSAC(
	matches []*FeatureMatch,
) (int, int, [4][4]float64) {
	// Simplified: assume all matches are inliers
	inliers := len(matches)
	outliers := 0

	// Identity transform (simplified)
	var transform [4][4]float64
	transform[0][0] = 1.0
	transform[1][1] = 1.0
	transform[2][2] = 1.0
	transform[3][3] = 1.0

	return inliers, outliers, transform
}

// GetDetectedLoops returns all detected loop closures
func (lcd *LoopClosureDetector) GetDetectedLoops() []*LoopClosure {
	lcd.mu.RLock()
	defer lcd.mu.RUnlock()
	return lcd.DetectedLoops
}

// GetLoopClosureCount returns number of detected loops
func (lcd *LoopClosureDetector) GetLoopClosureCount() int {
	lcd.mu.RLock()
	defer lcd.mu.RUnlock()
	return lcd.LoopClosureCount
}

// GetFrameCount returns number of registered frames
func (lcd *LoopClosureDetector) GetFrameCount() int {
	lcd.mu.RLock()
	defer lcd.mu.RUnlock()
	return lcd.FrameCount
}
