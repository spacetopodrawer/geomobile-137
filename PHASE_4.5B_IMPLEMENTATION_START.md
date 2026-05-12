# Phase 4.5B Implementation Start
**Date**: May 12, 2026 (3 days before official start)  
**Status**: READY FOR EXECUTION  
**Duration**: 12 weeks (May 15 - Aug 7, 2026)

---

## Current Baseline (Pre-Phase 4.5B)

### ✅ Verified Components
- Go backend: `v0.3.0-arcade-framework` compiled ✅
- Frontend: React/TypeScript operational ✅
- Database: SQLite schema ready ✅
- Config system: YAML parsing working ✅
- Documentation: Phase 4.5A complete ✅

### ❌ Missing for Phase 4.5B Week 1
The following GO implementations need to be created:

1. **pkg/sensors/rtk_integrator.go** (500 lines)
   - NTRIP protocol parser
   - Base station connection handling
   - RTK solution computation (±5cm target)
   
2. **pkg/sensors/camera_calibrator.go** (600 lines)
   - OpenCV intrinsic matrix calculation
   - Extrinsic calibration (camera pose)
   - Radial/tangential distortion correction
   
3. **pkg/sensors/imu_calibrator.go** (700 lines)
   - Accelerometer/gyroscope/magnetometer parsing
   - Bias removal and drift correction
   - Kalman filter for sensor fusion
   
4. **pkg/sensors/auto_calibrator.go** (400 lines)
   - Continuous calibration monitoring
   - Drift detection & alerting
   - Recalibration triggering

5. **pkg/photogrammetry/feature_detector.go** (500 lines)
   - SIFT/SURF implementation (for Week 2)
   - Dense feature extraction from image sequences

---

## Implementation Order (Week 1: May 15-21)

### Day 1-2: RTK Integration
**Owner**: Sensor Engineer  
**File**: `pkg/sensors/rtk_integrator.go`

```go
package sensors

import (
    "fmt"
    "net"
    "time"
    "cadastreia/pkg/model"
)

type RTKIntegrator struct {
    BaseStationHost string
    BaseStationPort int
    ConnectionPool  *net.Conn
    SolutionBuffer  chan *RTKSolution
}

type RTKSolution struct {
    Latitude      float64
    Longitude     float64
    Altitude      float64
    Accuracy      float64 // ±5cm target
    FixedSolution bool
    Timestamp     time.Time
}

func NewRTKIntegrator(host string, port int) *RTKIntegrator {
    return &RTKIntegrator{
        BaseStationHost: host,
        BaseStationPort: port,
        SolutionBuffer:  make(chan *RTKSolution, 100),
    }
}

func (r *RTKIntegrator) Connect() error {
    // NTRIP connection logic
    // Base station mount point negotiation
    // RTK correction stream initialization
    return nil
}

func (r *RTKIntegrator) ProcessRawMeasurements(rawData []byte) (*RTKSolution, error) {
    // Parse GNSS raw measurements
    // Apply RTK corrections
    // Compute fixed solution
    return &RTKSolution{}, nil
}

func (r *RTKIntegrator) VerifyAccuracy(solution *RTKSolution) bool {
    return solution.Accuracy <= 0.05 // 5cm threshold
}
```

### Day 3: Camera Calibration
**Owner**: CV Engineer  
**File**: `pkg/sensors/camera_calibrator.go`

```go
package sensors

import (
    "image"
    "cadastreia/pkg/model"
)

type CameraCalibrator struct {
    ImageSequence []image.Image
    CalibrationImages int // min 10-20 for good results
    IntrinsicMatrix [3][3]float64
    DistortionCoeffs [5]float64 // k1, k2, p1, p2, k3
}

func NewCameraCalibrator(calibImageCount int) *CameraCalibrator {
    return &CameraCalibrator{
        ImageSequence: make([]image.Image, 0, calibImageCount),
        CalibrationImages: calibImageCount,
    }
}

func (c *CameraCalibrator) AddCalibrationImage(img image.Image) error {
    if len(c.ImageSequence) >= c.CalibrationImages {
        return fmt.Errorf("calibration image limit reached")
    }
    c.ImageSequence = append(c.ImageSequence, img)
    return nil
}

func (c *CameraCalibrator) ComputeIntrinsicMatrix() error {
    // Detect chessboard corners in all images
    // Solve for intrinsic matrix (fx, fy, cx, cy, skew)
    // Compute distortion coefficients
    return nil
}

func (c *CameraCalibrator) ReprojectionError() float64 {
    // Return reprojection error (<0.5 pixels target)
    return 0.0
}
```

### Day 4-5: IMU Calibration
**Owner**: Sensor Engineer  
**File**: `pkg/sensors/imu_calibrator.go`

```go
package sensors

import (
    "time"
    "cadastreia/pkg/model"
)

type IMUCalibrator struct {
    Accelerometer [3]float64
    Gyroscope     [3]float64
    Magnetometer  [3]float64
    
    BiasAccel [3]float64
    BiasGyro  [3]float64
    
    DriftEstimate float64 // °/sec
    KalmanGain    float64
}

type KalmanState struct {
    Position    [3]float64
    Velocity    [3]float64
    Attitude    [3]float64
    BiasGyro    [3]float64
    Covariance  [15]float64
}

func NewIMUCalibrator() *IMUCalibrator {
    return &IMUCalibrator{
        KalmanGain: 0.01,
    }
}

func (i *IMUCalibrator) CalibrateBias(samples []IMUSample, duration time.Duration) error {
    // Collect static samples (device at rest)
    // Compute average as bias
    // Remove gravity component from accelerometer
    return nil
}

func (i *IMUCalibrator) EstimateDrift(samples []IMUSample) float64 {
    // Integrate gyroscope over time
    // Measure total rotation vs expected (0 at rest)
    // Return drift rate in °/sec
    return 0.0
}

func (i *IMUCalibrator) FuseSensors(kalman *KalmanState, imu *IMUSample) (*KalmanState, error) {
    // Predict step: integrate gyroscope
    // Update step: correct with accelerometer & magnetometer
    // Return updated state
    return kalman, nil
}
```

### Day 5-6: Auto-Calibration Framework
**Owner**: Calibration Engineer  
**File**: `pkg/sensors/auto_calibrator.go`

```go
package sensors

import (
    "time"
    "cadastreia/pkg/model"
)

type AutoCalibrator struct {
    RTK           *RTKIntegrator
    Camera        *CameraCalibrator
    IMU           *IMUCalibrator
    
    CalibrationInterval time.Duration
    DriftThreshold      float64
    LastCalibration     time.Time
}

func NewAutoCalibrator(rtk *RTKIntegrator, cam *CameraCalibrator, imu *IMUCalibrator) *AutoCalibrator {
    return &AutoCalibrator{
        RTK:                 rtk,
        Camera:              cam,
        IMU:                 imu,
        CalibrationInterval: 1 * time.Hour,
        DriftThreshold:      0.5, // °/sec
    }
}

func (a *AutoCalibrator) Monitor(ctx context.Context) error {
    ticker := time.NewTicker(time.Minute)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return nil
        case <-ticker.C:
            if a.ShouldRecalibrate() {
                if err := a.TriggerRecalibration(); err != nil {
                    // Log alert
                    return err
                }
            }
        }
    }
}

func (a *AutoCalibrator) ShouldRecalibrate() bool {
    elapsed := time.Since(a.LastCalibration)
    return elapsed > a.CalibrationInterval
}

func (a *AutoCalibrator) TriggerRecalibration() error {
    // Re-run all calibration steps
    // Compare with previous calibration
    // Alert if drift exceeds threshold
    a.LastCalibration = time.Now()
    return nil
}
```

---

## Testing Strategy (Week 1)

### RTK Tests
- Connection to base station: `test_rtk_connect()`
- Solution computation: `test_rtk_solution()`
- Accuracy verification: `test_rtk_accuracy()` (target: ±5cm @ 10Hz)

### Camera Tests
- Intrinsic matrix computation: `test_camera_intrinsic()`
- Reprojection error: `test_camera_reprojection_error()` (target: <0.5 pixels)
- Distortion correction: `test_camera_distortion_correction()`

### IMU Tests
- Bias estimation: `test_imu_bias_removal()`
- Drift calculation: `test_imu_drift()` (target: <0.1°/sec)
- Kalman fusion: `test_imu_kalman_fusion()`

### Integration Tests
- All three sensors in parallel
- Synchronization timing (<100ms latency)
- Cross-system compatibility (NEO-GEO, MAME, FBNeo)

---

## Go Dependencies to Add

```bash
go get github.com/go-echarts/go-echarts/v2  # For RTK plot visualization
go get gonum.org/v1/gonum/mat              # Matrix operations
go get gonum.org/v1/gonum/optimize         # Bundle adjustment
go get github.com/anthonynsimon/bild       # Image processing
```

---

## File Structure for Phase 4.5B

```
pkg/
├── sensors/                    # NEW - Week 1
│   ├── rtk_integrator.go
│   ├── camera_calibrator.go
│   ├── imu_calibrator.go
│   └── auto_calibrator.go
├── photogrammetry/             # NEW - Week 2
│   ├── feature_detector.go
│   ├── feature_matcher.go
│   ├── sfm_incremental.go
│   └── bundle_adjustment.go
├── recognition/                # NEW - Week 3
│   ├── symbol_recognizer.go
│   └── text_extractor.go
├── registry/                   # NEW - Week 4
│   ├── symbol_registry.go
│   └── sync_manager.go
├── llm/                        # NEW - Week 7
│   └── symbol_improver.go
├── consensus/                  # NEW - Week 8
│   └── consensus_engine.go
└── [existing packages]
```

---

## Success Criteria for Week 1

✅ All 4 files compiled and tested  
✅ RTK accuracy: ±5cm @ 10Hz  
✅ Camera reprojection error: <0.5 pixels  
✅ IMU drift: <0.1°/sec after calibration  
✅ Auto-calibration monitoring operational  
✅ Integration tests passing  
✅ All pull requests merged  

---

## Risks & Mitigations

| Risk | Mitigation |
|------|-----------|
| RTK initialization slow in urban canyons | Pre-position base station, use PPP fallback |
| Camera calibration requires 20+ high-quality images | Automate image capture with rotation instructions |
| IMU gyroscope drift accumulates | Use magnetometer fusion + gravity vector correction |
| Sensor timing synchronization | Use hardware timestamps, NTP sync for network |

---

## Timeline Checkpoint

- **May 12 (Today)**: Verify Phase 4.5A baseline, prepare Week 1 stubs
- **May 13-14**: Create sensor interface contracts
- **May 15 (Week 1 Start)**: Begin RTK integration
- **May 19**: Camera calibration complete
- **May 21**: All Week 1 tests passing
- **May 22 (Week 2 Start)**: Photogrammetry pipeline

---

## Next Steps

1. **Create `pkg/sensors/` directory structure**
2. **Stub out all 4 integrator files with package declarations**
3. **Add Go dependencies** with `go get`
4. **Create `cmd/test/sensors_test.go`** with integration test framework
5. **Begin RTK implementation** on May 15

This document will be updated as Phase 4.5B progresses.
