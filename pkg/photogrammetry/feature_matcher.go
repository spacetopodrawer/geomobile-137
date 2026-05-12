package photogrammetry

import (
	"fmt"
	"math"
	"sync"
)

// FeatureMatch represents a match between two keypoints
type FeatureMatch struct {
	SourceKeyPoint *KeyPoint
	TargetKeyPoint *KeyPoint
	Distance       float64
	Confidence     float64
}

// FeatureMatchResult contains matching statistics
type FeatureMatchResult struct {
	Matches      []*FeatureMatch
	MatchCount   int
	InlierCount  int
	OutlierCount int
	AverageError float64
	SuccessRate  float64
}

// FeatureMatcher matches keypoints between two images
type FeatureMatcher struct {
	mu sync.RWMutex

	DistanceThreshold float64
	RatioThreshold    float64
	MinMatchCount     int

	LastMatches *FeatureMatchResult
}

// NewFeatureMatcher creates a new feature matcher
func NewFeatureMatcher() *FeatureMatcher {
	return &FeatureMatcher{
		DistanceThreshold: 100.0,
		RatioThreshold:    0.75,
		MinMatchCount:     8,
	}
}

// MatchFeatures matches keypoints between two images
func (fm *FeatureMatcher) MatchFeatures(kpSource, kpTarget []*KeyPoint) (*FeatureMatchResult, error) {
	if len(kpSource) == 0 || len(kpTarget) == 0 {
		return nil, fmt.Errorf("empty keypoint arrays")
	}

	fm.mu.Lock()
	defer fm.mu.Unlock()

	matches := make([]*FeatureMatch, 0, len(kpSource))

	for _, src := range kpSource {
		best := &FeatureMatch{SourceKeyPoint: src, Distance: math.MaxFloat64}
		secondBest := &FeatureMatch{Distance: math.MaxFloat64}

		for _, tgt := range kpTarget {
			dist := fm.descriptorDistance(src.Descriptor, tgt.Descriptor)

			if dist < best.Distance {
				secondBest = best
				best = &FeatureMatch{
					SourceKeyPoint: src,
					TargetKeyPoint: tgt,
					Distance:       dist,
				}
			} else if dist < secondBest.Distance {
				secondBest.Distance = dist
			}
		}

		if best.Distance < fm.RatioThreshold*secondBest.Distance && best.Distance < fm.DistanceThreshold {
			best.Confidence = 1.0 - (best.Distance / fm.DistanceThreshold)
			matches = append(matches, best)
		}
	}

	minKPs := len(kpSource)
	if len(kpTarget) < minKPs {
		minKPs = len(kpTarget)
	}

	result := &FeatureMatchResult{
		Matches:      matches,
		MatchCount:   len(matches),
		InlierCount:  len(matches),
		OutlierCount: 0,
		AverageError: fm.computeAverageError(matches),
	}

	if minKPs > 0 {
		result.SuccessRate = float64(result.MatchCount) / float64(minKPs)
	}

	fm.LastMatches = result
	return result, nil
}

// descriptorDistance computes Euclidean distance between two descriptors
func (fm *FeatureMatcher) descriptorDistance(d1, d2 [128]float64) float64 {
	var sumSquares float64
	for i := 0; i < 128; i++ {
		diff := d1[i] - d2[i]
		sumSquares += diff * diff
	}
	return math.Sqrt(sumSquares)
}

// computeAverageError computes average descriptor distance of matches
func (fm *FeatureMatcher) computeAverageError(matches []*FeatureMatch) float64 {
	if len(matches) == 0 {
		return 0.0
	}

	var sum float64
	for _, match := range matches {
		sum += match.Distance
	}
	return sum / float64(len(matches))
}

// GetLastMatches returns the last computed match result
func (fm *FeatureMatcher) GetLastMatches() *FeatureMatchResult {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	return fm.LastMatches
}
