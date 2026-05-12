package photogrammetry

// Match represents a feature correspondence between two images
type Match struct {
	Feature1Index int     // Index in first image
	Feature2Index int     // Index in second image
	Distance      float64 // Descriptor distance (lower = better)
	RATIOScore    float64 // Lowe's ratio test score
}

// FeatureMatcher finds correspondences between feature sets
// Week 2: Part of Structure-from-Motion pipeline
type FeatureMatcher struct {
	LowesRatio     float64 // Default 0.7 (Lowe's ratio test threshold)
	MinMatches     int     // Minimum matches to accept pair
	RANSACIterations int   // RANSAC iterations for outlier removal
}

// NewFeatureMatcher creates a new feature matcher
func NewFeatureMatcher() *FeatureMatcher {
	return &FeatureMatcher{
		LowesRatio:     0.7,
		MinMatches:     20,
		RANSACIterations: 5000,
	}
}

// MatchFeatures finds correspondences between two feature sets
// Uses Lowe's ratio test for robustness
func (fm *FeatureMatcher) MatchFeatures(features1, features2 []*Feature) ([]*Match, error) {
	// TODO: Week 2 implementation
	// 1. For each feature in first image:
	//    - Find 2 nearest neighbors in second image (using descriptor distance)
	//    - Apply Lowe's ratio test: dist(best) / dist(2nd_best) < 0.7
	// 2. Return accepted matches
	return nil, nil
}

// FilterWithRANSAC removes outlier matches using RANSAC
// Fits fundamental matrix and keeps only inliers
func (fm *FeatureMatcher) FilterWithRANSAC(matches []*Match, features1, features2 []*Feature) ([]*Match, error) {
	// TODO: Week 2 implementation
	// 1. Repeatedly sample 8 random matches
	// 2. Compute fundamental matrix
	// 3. Count inliers (matches satisfying epipolar constraint)
	// 4. Return match with most inliers (target >95% inliers)
	return nil, nil
}
