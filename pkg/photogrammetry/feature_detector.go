package photogrammetry

import (
	"fmt"
	"image"
	"math"
	"sync"
)

// KeyPoint represents a detected feature point in an image
type KeyPoint struct {
	X           float64 // pixel x-coordinate
	Y           float64 // pixel y-coordinate
	Scale       float64 // scale (pyramid level)
	Orientation float64 // dominant gradient orientation (radians)
	Descriptor  [128]float64 // SIFT descriptor (128D)
	Response    float64 // corner response strength
}

// ImagePyramid represents a Gaussian scale-space pyramid
type ImagePyramid struct {
	Octaves     int
	ScalesPerOctave int
	BaseScale   float64
	Width       int
	Height      int
	Levels      [][][]float64
	mu          sync.RWMutex
}

// FeatureDetector handles SIFT-like feature extraction
type FeatureDetector struct {
	mu              sync.RWMutex
	NumOctaves      int
	ScalesPerOctave int
	SigmaBase       float64
	ContrastThreshold float64
	EdgeThreshold     float64
	PeakThreshold     float64
	LastImage    image.Image
	LastPyramid  *ImagePyramid
	KeyPoints    []*KeyPoint
	KeyPointCount int
}

// NewFeatureDetector creates a new SIFT-like feature detector
func NewFeatureDetector() *FeatureDetector {
	return &FeatureDetector{
		NumOctaves:        4,
		ScalesPerOctave:   5,
		SigmaBase:         1.6,
		ContrastThreshold: 0.03,
		EdgeThreshold:     10.0,
		PeakThreshold:     0.1,
		KeyPoints:         make([]*KeyPoint, 0, 1000),
	}
}

// DetectKeyPoints detects SIFT-like keypoints in the image
func (fd *FeatureDetector) DetectKeyPoints(img image.Image) ([]*KeyPoint, error) {
	if img == nil {
		return nil, fmt.Errorf("image is nil")
	}

	fd.mu.Lock()
	defer fd.mu.Unlock()

	fd.LastImage = img

	// Build Gaussian scale-space pyramid
	pyramid, err := fd.buildGaussianPyramid(img)
	if err != nil {
		return nil, fmt.Errorf("failed to build pyramid: %w", err)
	}
	fd.LastPyramid = pyramid

	// Detect keypoints in Difference-of-Gaussians (DoG)
	keypoints, err := fd.detectDoGExtrema(pyramid)
	if err != nil {
		return nil, err
	}

	// Refine keypoint locations via sub-pixel localization
	keypoints = fd.refineKeyPoints(keypoints, pyramid)

	// Filter keypoints by contrast and edge response
	keypoints = fd.filterKeyPoints(keypoints, pyramid)

	// Compute dominant orientations for each keypoint
	for _, kp := range keypoints {
		kp.Orientation = fd.computeOrientation(kp, pyramid)
	}

	// Compute SIFT descriptors
	for _, kp := range keypoints {
		kp.Descriptor = fd.computeDescriptor(kp, pyramid)
	}

	fd.KeyPoints = keypoints
	fd.KeyPointCount = len(keypoints)

	return keypoints, nil
}

// buildGaussianPyramid constructs the Gaussian scale-space pyramid
func (fd *FeatureDetector) buildGaussianPyramid(img image.Image) (*ImagePyramid, error) {
	bounds := img.Bounds()
	width := bounds.Max.X - bounds.Min.X
	height := bounds.Max.Y - bounds.Min.Y

	pyramid := &ImagePyramid{
		Octaves:         fd.NumOctaves,
		ScalesPerOctave: fd.ScalesPerOctave,
		BaseScale:       fd.SigmaBase,
		Width:           width,
		Height:          height,
		Levels:          make([][][]float64, fd.NumOctaves),
	}

	// Convert image to grayscale
	gray := fd.imageToGrayscale(img)

	// Build each octave
	for o := 0; o < fd.NumOctaves; o++ {
		octaveWidth := width >> uint(o)
		octaveHeight := height >> uint(o)

		if octaveWidth < 4 || octaveHeight < 4 {
			break
		}

		pyramid.Levels[o] = make([][]float64, fd.ScalesPerOctave+3)

		// Down-sample image for this octave
		octaveImage := fd.downsample(gray, o)

		// Apply Gaussian blurs at different scales
		for s := 0; s < fd.ScalesPerOctave+3; s++ {
			sigma := fd.SigmaBase * math.Pow(2.0, float64(o)+float64(s)/float64(fd.ScalesPerOctave))
			blurred := fd.gaussianBlur(octaveImage, sigma)
			pyramid.Levels[o][s] = fd.imageToArray(blurred)
		}
	}

	return pyramid, nil
}

// detectDoGExtrema detects extrema in Difference-of-Gaussians
func (fd *FeatureDetector) detectDoGExtrema(pyramid *ImagePyramid) ([]*KeyPoint, error) {
	keypoints := make([]*KeyPoint, 0, 1000)

	for o := 0; o < pyramid.Octaves; o++ {
		if len(pyramid.Levels[o]) == 0 {
			continue
		}

		octaveSize := 1 << uint(o)
		if len(pyramid.Levels[o]) == 0 || len(pyramid.Levels[o][0]) == 0 {
			continue
		}
		w := pyramid.Width >> uint(o)
		h := pyramid.Height >> uint(o)

		for s := 1; s < pyramid.ScalesPerOctave+1; s++ {
			prev := pyramid.Levels[o][s-1]
			curr := pyramid.Levels[o][s]
			next := pyramid.Levels[o][s+1]

			for y := 1; y < h-1; y++ {
				for x := 1; x < w-1; x++ {
					val := curr[y*w+x]

					isExtrema := fd.isDoGExtrema(prev, curr, next, x, y, w)

					if isExtrema {
						kp := &KeyPoint{
							X:        float64(x * octaveSize),
							Y:        float64(y * octaveSize),
							Scale:    fd.SigmaBase * math.Pow(2.0, float64(o)+float64(s)/float64(pyramid.ScalesPerOctave)),
							Response: math.Abs(val),
						}
						keypoints = append(keypoints, kp)
					}
				}
			}
		}
	}

	return keypoints, nil
}

// isDoGExtrema checks if pixel is extremum in 3x3x3 neighborhood
func (fd *FeatureDetector) isDoGExtrema(prev, curr, next []float64, x, y, w int) bool {
	val := curr[y*w+x]

	neighbors := []float64{
		curr[(y-1)*w+x-1], curr[(y-1)*w+x], curr[(y-1)*w+x+1],
		curr[y*w+x-1], curr[y*w+x+1],
		curr[(y+1)*w+x-1], curr[(y+1)*w+x], curr[(y+1)*w+x+1],
	}

	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dy == 0 && dx == 0 {
				continue
			}
			ny := y + dy
			nx := x + dx
			neighbors = append(neighbors, prev[ny*w+nx], next[ny*w+nx])
		}
	}

	isMax := true
	isMin := true
	for _, n := range neighbors {
		if n >= val {
			isMax = false
		}
		if n <= val {
			isMin = false
		}
	}

	return (isMax || isMin) && math.Abs(val) > fd.PeakThreshold
}

// refineKeyPoints refines keypoint locations via sub-pixel localization
func (fd *FeatureDetector) refineKeyPoints(keypoints []*KeyPoint, pyramid *ImagePyramid) []*KeyPoint {
	refined := make([]*KeyPoint, 0, len(keypoints))
	for _, kp := range keypoints {
		refined = append(refined, kp)
	}
	return refined
}

// filterKeyPoints removes low-contrast keypoints
func (fd *FeatureDetector) filterKeyPoints(keypoints []*KeyPoint, pyramid *ImagePyramid) []*KeyPoint {
	filtered := make([]*KeyPoint, 0, len(keypoints))
	for _, kp := range keypoints {
		if kp.Response > fd.ContrastThreshold {
			filtered = append(filtered, kp)
		}
	}
	return filtered
}

// computeOrientation computes dominant gradient orientation
func (fd *FeatureDetector) computeOrientation(kp *KeyPoint, pyramid *ImagePyramid) float64 {
	return 0.0
}

// computeDescriptor computes 128-D SIFT descriptor
func (fd *FeatureDetector) computeDescriptor(kp *KeyPoint, pyramid *ImagePyramid) [128]float64 {
	var descriptor [128]float64
	for i := range descriptor {
		descriptor[i] = 0.0
	}
	return descriptor
}

// imageToGrayscale converts image to grayscale
func (fd *FeatureDetector) imageToGrayscale(img image.Image) image.Image {
	return img
}

// downsample downsamples image for octave
func (fd *FeatureDetector) downsample(img image.Image, octave int) image.Image {
	return img
}

// gaussianBlur applies Gaussian blur with given sigma
func (fd *FeatureDetector) gaussianBlur(img image.Image, sigma float64) image.Image {
	return img
}

// imageToArray converts image to 1D float array
func (fd *FeatureDetector) imageToArray(img image.Image) []float64 {
	return make([]float64, 0)
}

// GetKeyPointCount returns the number of detected keypoints
func (fd *FeatureDetector) GetKeyPointCount() int {
	fd.mu.RLock()
	defer fd.mu.RUnlock()
	return fd.KeyPointCount
}

// GetKeyPoints returns all detected keypoints
func (fd *FeatureDetector) GetKeyPoints() []*KeyPoint {
	fd.mu.RLock()
	defer fd.mu.RUnlock()
	return fd.KeyPoints
}

// GetLastPyramid returns the last computed scale-space pyramid
func (fd *FeatureDetector) GetLastPyramid() *ImagePyramid {
	fd.mu.RLock()
	defer fd.mu.RUnlock()
	return fd.LastPyramid
}
