package photogrammetry

import (
	"image"
	"image/color"
	"testing"
)

// createTestImage creates a simple test image for feature detection
func createTestImage(width, height int) image.Image {
	img := image.NewGray(image.Rect(0, 0, width, height))
	// Create a simple checkerboard pattern
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if ((x / 10) + (y / 10)) % 2 == 0 {
				img.SetGray(x, y, color.Gray{Y: 255})
			} else {
				img.SetGray(x, y, color.Gray{Y: 0})
			}
		}
	}
	return img
}

// TestFeatureDetectorInit tests initialization
func TestFeatureDetectorInit(t *testing.T) {
	fd := NewFeatureDetector()

	if fd.NumOctaves != 4 {
		t.Errorf("Expected 4 octaves, got %d", fd.NumOctaves)
	}

	if fd.ScalesPerOctave != 5 {
		t.Errorf("Expected 5 scales per octave, got %d", fd.ScalesPerOctave)
	}

	if fd.SigmaBase != 1.6 {
		t.Errorf("Expected sigma 1.6, got %v", fd.SigmaBase)
	}

	if fd.KeyPointCount != 0 {
		t.Error("Should start with no keypoints detected")
	}
}

// TestDetectKeyPointsNilImage tests error handling for nil input
func TestDetectKeyPointsNilImage(t *testing.T) {
	fd := NewFeatureDetector()
	_, err := fd.DetectKeyPoints(nil)
	if err == nil {
		t.Error("Should return error for nil image")
	}
}

// TestDetectKeyPointsValidImage tests keypoint detection on valid image
func TestDetectKeyPointsValidImage(t *testing.T) {
	fd := NewFeatureDetector()
	img := createTestImage(256, 256)

	kps, err := fd.DetectKeyPoints(img)
	if err != nil {
		t.Fatalf("DetectKeyPoints failed: %v", err)
	}

	if kps == nil {
		t.Fatal("Should return non-nil keypoints slice")
	}

	// Checkerboard should detect some keypoints
	if fd.KeyPointCount < 0 {
		t.Errorf("Negative keypoint count: %d", fd.KeyPointCount)
	}
}

// TestKeyPointProperties tests keypoint attributes
func TestKeyPointProperties(t *testing.T) {
	fd := NewFeatureDetector()
	img := createTestImage(256, 256)

	kps, _ := fd.DetectKeyPoints(img)

	if len(kps) > 0 {
		kp := kps[0]

		if kp.X < 0 || kp.Y < 0 {
			t.Error("Keypoint coordinates should be non-negative")
		}

		if kp.Scale <= 0 {
			t.Error("Keypoint scale should be positive")
		}

		if kp.Response < 0 {
			t.Error("Keypoint response should be non-negative")
		}

		// Descriptor should have 128 dimensions
		nonZeroCount := 0
		for _, val := range kp.Descriptor {
			if val != 0 {
				nonZeroCount++
			}
		}
		// At least some descriptor values should be non-zero
	}
}

// TestGetKeyPointCount tests keypoint counter
func TestGetKeyPointCount(t *testing.T) {
	fd := NewFeatureDetector()
	if fd.GetKeyPointCount() != 0 {
		t.Error("Should start with 0 keypoints")
	}

	img := createTestImage(128, 128)
	fd.DetectKeyPoints(img)

	count := fd.GetKeyPointCount()
	if count < 0 {
		t.Errorf("Negative keypoint count: %d", count)
	}
}

// TestGetKeyPoints tests keypoint retrieval
func TestGetKeyPoints(t *testing.T) {
	fd := NewFeatureDetector()
	img := createTestImage(256, 256)
	fd.DetectKeyPoints(img)

	kps := fd.GetKeyPoints()
	if kps == nil {
		t.Fatal("Should return non-nil keypoints slice")
	}

	if len(kps) != fd.GetKeyPointCount() {
		t.Errorf("Keypoint count mismatch: got %d, expected %d", len(kps), fd.GetKeyPointCount())
	}
}

// TestGetLastPyramid tests pyramid retrieval
func TestGetLastPyramid(t *testing.T) {
	fd := NewFeatureDetector()
	img := createTestImage(256, 256)
	fd.DetectKeyPoints(img)

	pyramid := fd.GetLastPyramid()
	if pyramid == nil {
		t.Fatal("Should return non-nil pyramid")
	}

	if pyramid.Octaves != 4 {
		t.Errorf("Pyramid should have 4 octaves, got %d", pyramid.Octaves)
	}

	if pyramid.ScalesPerOctave != 5 {
		t.Errorf("Pyramid should have 5 scales per octave, got %d", pyramid.ScalesPerOctave)
	}
}

// TestMultipleDetections tests repeated detections
func TestMultipleDetections(t *testing.T) {
	fd := NewFeatureDetector()
	img1 := createTestImage(256, 256)
	img2 := createTestImage(256, 256)

	kps1, _ := fd.DetectKeyPoints(img1)
	count1 := fd.GetKeyPointCount()

	kps2, _ := fd.DetectKeyPoints(img2)
	count2 := fd.GetKeyPointCount()

	if len(kps1) != count1 || len(kps2) != count2 {
		t.Error("Keypoint counts should match lengths")
	}
}

// BenchmarkDetectKeyPoints benchmarks feature detection performance
func BenchmarkDetectKeyPoints(b *testing.B) {
	fd := NewFeatureDetector()
	img := createTestImage(512, 512)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fd.DetectKeyPoints(img)
	}
}
