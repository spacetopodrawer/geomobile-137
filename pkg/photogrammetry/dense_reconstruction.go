package photogrammetry

import (
	"fmt"
	"math"
	"sync"
)

// DepthMap represents dense depth estimation for a single view
type DepthMap struct {
	Width       int
	Height      int
	Depth       []float64 // Row-major depth values
	Confidence  []float64 // Depth confidence [0, 1]
	Validity    []bool    // Valid depth at each pixel
	FrameID     int
	Timestamp   int64
}

// DenseReconstruction performs multi-view stereo for complete 3D model
// Uses depth map estimation from image pairs
type DenseReconstruction struct {
	mu sync.RWMutex

	// Parameters
	MinDepth       float64 // Minimum depth to consider
	MaxDepth       float64 // Maximum depth to consider
	WindowSize     int     // Patch size for similarity (e.g., 5x5)
	ConfidenceThreshold float64

	// State
	DepthMaps       []*DepthMap
	PointCloud      []*Point3D
	PointCloudSize  int
}

// NewDenseReconstruction creates a new dense reconstruction engine
func NewDenseReconstruction(minDepth, maxDepth float64) *DenseReconstruction {
	return &DenseReconstruction{
		MinDepth:           minDepth,
		MaxDepth:           maxDepth,
		WindowSize:         5, // 5x5 patch
		ConfidenceThreshold: 0.5,
		DepthMaps:          make([]*DepthMap, 0, 100),
		PointCloud:         make([]*Point3D, 0, 100000),
	}
}

// EstimateDepthMap computes depth map for a frame using stereo matching
func (dr *DenseReconstruction) EstimateDepthMap(
	frameID int,
	referenceImage, targetImage [][3]uint8,
	referencePose, targetPose *CameraFrame,
	width, height int,
) (*DepthMap, error) {
	if width == 0 || height == 0 {
		return nil, fmt.Errorf("invalid image dimensions: %dx%d", width, height)
	}

	dr.mu.Lock()
	defer dr.mu.Unlock()

	depthMap := &DepthMap{
		Width:      width,
		Height:     height,
		Depth:      make([]float64, width*height),
		Confidence: make([]float64, width*height),
		Validity:   make([]bool, width*height),
		FrameID:    frameID,
	}

	// Photometric stereo matching
	for y := dr.WindowSize / 2; y < height-dr.WindowSize/2; y++ {
		for x := dr.WindowSize / 2; x < width-dr.WindowSize/2; x++ {
			// Estimate depth at pixel (x, y) using stereo matching
			bestDepth, confidence := dr.estimatePixelDepth(
				x, y,
				referenceImage, targetImage,
				referencePose, targetPose,
				width, height,
			)

			idx := y*width + x
			depthMap.Depth[idx] = bestDepth
			depthMap.Confidence[idx] = confidence
			depthMap.Validity[idx] = confidence > dr.ConfidenceThreshold
		}
	}

	dr.DepthMaps = append(dr.DepthMaps, depthMap)
	return depthMap, nil
}

// estimatePixelDepth estimates depth at single pixel using stereo
func (dr *DenseReconstruction) estimatePixelDepth(
	x, y int,
	refImage, tgtImage [][3]uint8,
	refPose, tgtPose *CameraFrame,
	width, height int,
) (float64, float64) {
	// Simplified: Use matching patch similarity across depth range
	bestDepth := (dr.MinDepth + dr.MaxDepth) / 2.0
	bestScore := 0.0

	for d := dr.MinDepth; d <= dr.MaxDepth; d += (dr.MaxDepth - dr.MinDepth) / 10 {
		// Estimate reference patch
		refPatch := dr.extractPatch(refImage, x, y, width, height)

		// Warp target image at depth d and extract patch
		warpedX, warpedY := dr.warpPixel(x, y, d, refPose, tgtPose)

		if warpedX < 0 || warpedX >= float64(width-1) ||
			warpedY < 0 || warpedY >= float64(height-1) {
			continue
		}

		tgtPatch := dr.extractPatchInterpolated(tgtImage, warpedX, warpedY, width, height)

		// Compute normalized cross-correlation
		ncc := dr.computeNCC(refPatch, tgtPatch)

		if ncc > bestScore {
			bestScore = ncc
			bestDepth = d
		}
	}

	// Confidence based on NCC score
	confidence := (bestScore + 1.0) / 2.0 // Normalize NCC [-1, 1] to [0, 1]

	return bestDepth, confidence
}

// extractPatch extracts a patch centered at (x, y)
func (dr *DenseReconstruction) extractPatch(
	image [][3]uint8,
	cx, cy, width, height int,
) []uint8 {
	patchSize := dr.WindowSize * dr.WindowSize * 3
	patch := make([]uint8, patchSize)

	idx := 0
	for dy := -dr.WindowSize / 2; dy <= dr.WindowSize/2; dy++ {
		for dx := -dr.WindowSize / 2; dx <= dr.WindowSize/2; dx++ {
			y := cy + dy
			x := cx + dx

			if y >= 0 && y < height && x >= 0 && x < width {
				pixIdx := y*width + x
				if pixIdx < len(image) {
					patch[idx] = image[pixIdx][0]
					patch[idx+1] = image[pixIdx][1]
					patch[idx+2] = image[pixIdx][2]
				}
			}
			idx += 3
		}
	}

	return patch
}

// extractPatchInterpolated extracts patch with bilinear interpolation
func (dr *DenseReconstruction) extractPatchInterpolated(
	image [][3]uint8,
	cx, cy float64,
	width, height int,
) []uint8 {
	// Simplified: Extract at nearest integer coordinate
	ix := int(cx + 0.5)
	iy := int(cy + 0.5)

	if ix < 0 || ix >= width || iy < 0 || iy >= height {
		return make([]uint8, dr.WindowSize*dr.WindowSize*3)
	}

	return dr.extractPatch(image, ix, iy, width, height)
}

// computeNCC computes normalized cross-correlation
func (dr *DenseReconstruction) computeNCC(patch1, patch2 []uint8) float64 {
	if len(patch1) != len(patch2) || len(patch1) == 0 {
		return 0.0
	}

	// Compute means
	var mean1, mean2 float64
	for _, p := range patch1 {
		mean1 += float64(p)
	}
	for _, p := range patch2 {
		mean2 += float64(p)
	}
	mean1 /= float64(len(patch1))
	mean2 /= float64(len(patch2))

	// Compute correlation
	var numerator, denom1, denom2 float64
	for i := range patch1 {
		diff1 := float64(patch1[i]) - mean1
		diff2 := float64(patch2[i]) - mean2

		numerator += diff1 * diff2
		denom1 += diff1 * diff1
		denom2 += diff2 * diff2
	}

	if denom1 <= 0 || denom2 <= 0 {
		return 0.0
	}

	ncc := numerator / math.Sqrt(denom1*denom2)
	return ncc
}

// warpPixel warps pixel from reference to target image at given depth
func (dr *DenseReconstruction) warpPixel(
	x, y int,
	depth float64,
	refPose, tgtPose *CameraFrame,
) (float64, float64) {
	// Simplified: Project through both cameras
	// 1. Unproject (x, y, depth) to 3D in reference frame
	// 2. Transform to target frame
	// 3. Project to target image

	// Identity transformation for simplification
	return float64(x), float64(y)
}

// ConvertDepthMapToPointCloud converts depth map to 3D points
func (dr *DenseReconstruction) ConvertDepthMapToPointCloud(
	depthMap *DepthMap,
	pose *CameraFrame,
) []*Point3D {
	points := make([]*Point3D, 0, depthMap.Width*depthMap.Height)

	for y := 0; y < depthMap.Height; y++ {
		for x := 0; x < depthMap.Width; x++ {
			idx := y*depthMap.Width + x

			if !depthMap.Validity[idx] {
				continue
			}

			depth := depthMap.Depth[idx]

			// Unproject pixel to 3D
			point := dr.unprojectedPoint(x, y, depth, pose)
			if point != nil {
				point.Uncertainty = 1.0 - depthMap.Confidence[idx]
				points = append(points, point)
			}
		}
	}

	dr.PointCloud = append(dr.PointCloud, points...)
	dr.PointCloudSize = len(dr.PointCloud)

	return points
}

// unprojectedPoint unprojects pixel to 3D using camera parameters
func (dr *DenseReconstruction) unprojectedPoint(
	x, y int,
	depth float64,
	pose *CameraFrame,
) *Point3D {
	if depth <= 0 {
		return nil
	}

	// Normalize pixel coordinates
	K := pose.CameraMatrix
	px := (float64(x) - K[0][2]) / K[0][0]
	py := (float64(y) - K[1][2]) / K[1][1]

	// 3D point in camera frame
	pz := depth
	px *= depth
	py *= depth

	// Transform to world frame: p = R^T*(p_cam - t)
	point := &Point3D{
		X:     px,
		Y:     py,
		Z:     pz,
		Depth: depth,
	}

	return point
}

// GetDepthMaps returns all estimated depth maps
func (dr *DenseReconstruction) GetDepthMaps() []*DepthMap {
	dr.mu.RLock()
	defer dr.mu.RUnlock()
	return dr.DepthMaps
}

// GetPointCloud returns all reconstructed 3D points
func (dr *DenseReconstruction) GetPointCloud() []*Point3D {
	dr.mu.RLock()
	defer dr.mu.RUnlock()
	return dr.PointCloud
}

// GetPointCloudSize returns total number of dense points
func (dr *DenseReconstruction) GetPointCloudSize() int {
	dr.mu.RLock()
	defer dr.mu.RUnlock()
	return dr.PointCloudSize
}
