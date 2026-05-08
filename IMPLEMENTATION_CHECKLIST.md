# Document 7: Implementation Checklist
## 12-Week Phase 4.5B Sprint Plan with Weekly Milestones

**Phase**: 4.5B (Universal Save Format & Adaptive Rendering)  
**Duration**: 12 weeks (May 15 - Aug 7, 2026)  
**Team Size**: 6-8 engineers  
**Document Version**: v1.0  
**Date**: May 8, 2026

---

## Executive Summary

This sprint plan executes all 13 layers of Cadastre_IA Phase 4.5B within 12 weeks:

**Weeks 1-3**: Layers 0-2 (Sensors, Calibration, Photogrammetry)  
**Weeks 4-6**: Layers 3-6 (Recognition, Extraction, Registry, Transport)  
**Weeks 7-9**: Layers 7-10 (LLM, Variants, Consensus, QA)  
**Weeks 10-12**: Layers 11-13 (Output, Feedback, Archival) + Integration & Release

**Key Milestones**:
- Week 3: Photogrammetry pipeline working (end-to-end 3D reconstruction)
- Week 6: P2P sync operational (objects synchronized across platforms)
- Week 9: Consensus algorithm live (symbols improving automatically)
- Week 12: v0.4.0 release (production-ready universal save format)

---

## Week-by-Week Breakdown

### Week 1: Sensor Integration & Calibration Setup

**Goal**: RTK/GNSS + IMU + Camera calibration pipeline operational

**Tasks**:
1. **RTK/GNSS Integration** (Layer 0)
   - Implement RTK parser (NTRIP protocol)
   - Configure base station connection
   - Test accuracy: target ±5cm
   - Owner: Sensor Engineer A
   - Deliverable: rtk_integrator.go (500 lines)

2. **Camera Calibration** (Layer 1)
   - Implement intrinsic matrix calibration (OpenCV)
   - Extrinsic calibration (camera pose estimation)
   - Radial/tangential distortion correction
   - Owner: Computer Vision Engineer
   - Deliverable: camera_calibrator.go (600 lines)

3. **IMU Integration** (Layer 0-1)
   - Implement accelerometer/gyroscope/magnetometer parsing
   - Bias removal, drift correction
   - Sensor fusion (Kalman filter initial)
   - Owner: Sensor Engineer B
   - Deliverable: imu_calibrator.go (700 lines)

4. **Auto-Calibration Framework** (Layer 1)
   - Implement continuous calibration monitoring
   - Detect and alert on calibration drift
   - Owner: Calibration Engineer
   - Deliverable: auto_calibrator.go (400 lines)

**Testing**:
- RTK accuracy test: ±5cm @ 10Hz
- Camera calibration error: <0.5 pixel reprojection
- IMU drift: <0.1°/sec after calibration

**Blockers**: None (all libraries available)  
**Risks**: RTK initialization slow in urban canyons → mitigation: pre-position base station

**Status**: PLANNED

---

### Week 2: Photogrammetry Foundation (SfM)

**Goal**: Structure-from-Motion pipeline processing image sequences

**Tasks**:
1. **Feature Detection** (Layer 2)
   - SIFT/SURF implementation (OpenCV/VLFeat)
   - Dense feature extraction from image sequences
   - Owner: CV Engineer
   - Deliverable: feature_detector.go (500 lines)

2. **Feature Matching** (Layer 2)
   - Lowe's ratio test for good matches
   - RANSAC outlier removal (>95% inliers)
   - Owner: CV Engineer
   - Deliverable: feature_matcher.go (600 lines)

3. **Incremental SfM** (Layer 2)
   - Initial pair selection (best baseline)
   - Triangulation (DLT method)
   - Incremental pose estimation
   - Owner: 3D Reconstruction Engineer
   - Deliverable: sfm_incremental.go (800 lines)

4. **Bundle Adjustment** (Layer 2)
   - Ceres Solver integration
   - Reprojection error minimization
   - Convergence monitoring
   - Owner: Optimization Engineer
   - Deliverable: bundle_adjuster.go (700 lines)

**Testing**:
- SfM accuracy: <2cm RMS reprojection error
- Bundle adjustment convergence: <0.5 pixel residual
- Throughput: 100 images → sparse point cloud in <60 seconds

**Blockers**: None  
**Risks**: Ceres Solver integration complexity → use proven configurations

**Status**: PLANNED

---

### Week 3: Dense Reconstruction & Mesh

**Goal**: Dense point cloud → mesh generation (end-to-end)

**Tasks**:
1. **Multi-View Stereo** (Layer 2)
   - MVS reconstruction from sparse point cloud
   - Depth map fusion from multiple views
   - Owner: CV Engineer
   - Deliverable: mvs_reconstructor.go (600 lines)

2. **Mesh Generation** (Layer 2)
   - Delaunay/Poisson reconstruction
   - Surface smoothing, cleanup
   - Owner: 3D Engineer
   - Deliverable: mesh_generator.go (500 lines)

3. **Texture Mapping** (Layer 2)
   - Texture atlas generation
   - Seam blending
   - Owner: Graphics Engineer
   - Deliverable: texture_mapper.go (400 lines)

4. **RTK Ground Control Integration** (Layer 2)
   - Register point cloud to RTK coordinates
   - Apply RTK-to-mesh alignment
   - Owner: Calibration Engineer
   - Deliverable: rtk_registration.go (300 lines)

5. **End-to-End Test** (Layer 2)
   - Images → Point cloud → Mesh → SVG extraction
   - Owner: Integration Engineer
   - Deliverable: photogrammetry_e2e_test.go (400 lines)

**Testing**:
- Dense reconstruction: >100 points/m² density
- Mesh coverage: >95% of scene
- Texture quality: Seamless blending
- RTK alignment: ±5cm accuracy maintained

**Blockers**: MVS library selection (CloudCompare vs. COLMAP vs. OpenMVG)  
**Risks**: Memory intensive for large point clouds → stream processing

**Status**: PLANNED

**MILESTONE 1 COMPLETE**: Photogrammetry pipeline working end-to-end (Weeks 1-3)

---

### Week 4: Recognition & Object Detection

**Goal**: Scene understanding (detect buildings, parcels, utilities, boundaries)

**Tasks**:
1. **Object Detection** (Layer 3)
   - YOLOv8/Faster R-CNN for building/parcel detection
   - Real-time inference (<100ms per image)
   - Owner: ML Engineer
   - Deliverable: object_detector.go (400 lines)

2. **OCR Integration** (Layer 3)
   - Tesseract/PaddleOCR for text extraction
   - Extract street signs, building numbers, labels
   - Owner: ML Engineer
   - Deliverable: ocr_engine.go (300 lines)

3. **Semantic Segmentation** (Layer 3)
   - DeepLabv3+ for pixel-level classification
   - Classify each pixel (building/ground/vegetation/etc.)
   - Owner: ML Engineer
   - Deliverable: semantic_segmenter.go (400 lines)

4. **Motion Tracking** (Layer 3)
   - Track moving objects between frames
   - Detect dynamic vs. static objects
   - Owner: CV Engineer
   - Deliverable: motion_tracker.go (300 lines)

**Testing**:
- Object detection: >90% precision/recall on test set
- OCR: >95% character accuracy
- Segmentation: >85% IoU on test imagery

**Blockers**: ML model licenses  
**Risks**: GPU availability for inference → consider ONNX CPU inference fallback

**Status**: PLANNED

---

### Week 5: Attribute Extraction & SVG Generation

**Goal**: Convert geometric data → minified SVG (Layer 4)

**Tasks**:
1. **Attribute Extraction** (Layer 4)
   - Extract semantic attributes (owner, area, type, material)
   - Link attributes to geometry
   - Owner: Data Engineer
   - Deliverable: attribute_extractor.go (400 lines)

2. **SVG Minification** (Layer 4)
   - Minify SVG path data (remove whitespace, reduce precision)
   - Apply dictionary encoding patterns
   - Owner: Data Engineer
   - Deliverable: svg_minifier.go (300 lines)

3. **3-Layer Compression** (Layer 4)
   - Minification → Dictionary → DEFLATE
   - Achieve 93% size reduction
   - Owner: Compression Engineer
   - Deliverable: svg_compressor.go (400 lines)

4. **Decompression** (Layer 4)
   - Fast decompression (<10ms per sprite)
   - Reverse pipeline
   - Owner: Compression Engineer
   - Deliverable: svg_decompressor.go (300 lines)

5. **Compression Metadata** (Layer 4)
   - Track compression ratios, timing
   - Generate compression_metadata table records
   - Owner: Data Engineer
   - Deliverable: compression_recorder.go (200 lines)

**Testing**:
- Compression ratio: 93% (2,048 bytes → 128 bytes)
- Decompression: <10ms per object
- Accuracy: Lossless (decompressed ≡ original SVG)

**Blockers**: None  
**Risks**: DEFLATE efficiency varies by input → benchmark on diverse objects

**Status**: PLANNED

---

### Week 6: Registry & P2P Transport

**Goal**: Store objects in registry, implement P2P sync (Layers 5-6)

**Tasks**:
1. **Object Registry** (Layer 5)
   - Implement registry_objects table schema
   - CRUD operations for objects
   - Versioning support
   - Owner: Backend Engineer A
   - Deliverable: registry_service.go (800 lines)

2. **P2P WebSocket Sync** (Layer 6)
   - WebSocket handshake protocol
   - Message framing + checksum validation
   - SYNC_REQUEST/SYNC_RESPONSE implementation
   - Owner: Network Engineer
   - Deliverable: p2p_sync.go (800 lines)

3. **Delta Encoding** (Layer 6)
   - VCDIFF binary diff implementation
   - SYNC_DELTA message support
   - Owner: Compression Engineer
   - Deliverable: delta_encoder.go (400 lines)

4. **Conflict Resolution** (Layer 6)
   - Vector clock ordering
   - SYNC_CONFLICT message handling
   - Owner: Distributed Systems Engineer
   - Deliverable: conflict_resolver.go (500 lines)

5. **Offline Queuing** (Layer 6)
   - Queue updates when offline
   - Resync on reconnection
   - Owner: Network Engineer
   - Deliverable: offline_queue.go (300 lines)

6. **Integration Test** (Layer 5-6)
   - Multi-client sync test
   - Verify eventual consistency
   - Owner: Integration Engineer
   - Deliverable: sync_integration_test.go (600 lines)

**Testing**:
- Sync latency: <50ms LAN, <100ms WAN
- Compression ratio: 70-90% with delta
- Throughput: 1,000+ objects/sec
- Conflict resolution: All scenarios handled

**Blockers**: None  
**Risks**: WebSocket connection drops under load → implement heartbeat/reconnect logic

**Status**: PLANNED

**MILESTONE 2 COMPLETE**: P2P sync operational (Weeks 4-6)

---

### Week 7: LLM Decoding & Personalization

**Goal**: Implement LLM-driven adaptive rendering (Layer 7)

**Tasks**:
1. **LLM Integration** (Layer 7)
   - Claude API integration (batching, error handling)
   - Prompt template system
   - Owner: ML Engineer
   - Deliverable: llm_client.go (400 lines)

2. **Platform Detection** (Layer 7)
   - Detect device capabilities (resolution, colors, etc.)
   - Load platform_capabilities from DB
   - Owner: Backend Engineer B
   - Deliverable: platform_detector.go (300 lines)

3. **Prompt Generation** (Layer 7)
   - Build prompts from templates (Document 3)
   - Context assembly (object, platform, user)
   - Owner: Data Engineer
   - Deliverable: prompt_builder.go (400 lines)

4. **Rendering Hints Cache** (Layer 7)
   - Cache LLM responses (Redis)
   - Invalidate on object update
   - Owner: Cache Engineer
   - Deliverable: hints_cache.go (300 lines)

5. **Personalization** (Layer 8)
   - Load user profile (accessibility, preferences, skill)
   - Generate personalized variant
   - Owner: Data Engineer
   - Deliverable: personalization_engine.go (400 lines)

**Testing**:
- LLM latency: <100ms per prompt
- Cache hit rate: >70%
- Rendering hint accuracy: All platforms produce valid output

**Blockers**: Claude API rate limits → implement queuing  
**Risks**: LLM cost at scale → caching critical

**Status**: PLANNED

---

### Week 8: Variants & User Feedback

**Goal**: Per-user rendering variants, feedback collection (Layers 8-9)

**Tasks**:
1. **Variant Storage** (Layer 8)
   - Implement object_variants table
   - CRUD for variants
   - Owner: Backend Engineer A
   - Deliverable: variant_service.go (500 lines)

2. **Variant Cache** (Layer 8)
   - Pre-render popular variants
   - Cache management (TTL, invalidation)
   - Owner: Cache Engineer
   - Deliverable: variant_prerender.go (400 lines)

3. **Feedback Collection** (Layer 9)
   - Implement feedback form UI
   - Collect implicit feedback (dwell time, interactions)
   - Store in user_feedback table
   - Owner: Frontend Engineer
   - Deliverable: feedback_collector.go (400 lines)

4. **Feedback Normalization** (Layer 9)
   - Convert implicit → explicit ratings
   - Apply user weighting
   - Owner: Data Engineer
   - Deliverable: feedback_normalizer.go (300 lines)

5. **Consensus Input Pipeline** (Layer 9)
   - Stream feedback to consensus layer
   - Real-time consensus monitoring
   - Owner: Data Engineer
   - Deliverable: consensus_input.go (300 lines)

**Testing**:
- Variant generation: <500ms per user+platform combo
- Cache hit rate: >70% for popular objects
- Feedback ingestion: 100,000 items/minute

**Blockers**: None  
**Risks**: Feedback quality → implement review queue for edge cases

**Status**: PLANNED

---

### Week 9: Consensus Algorithm Live

**Goal**: Automatic baseline evolution (v1.0 → v1.1 → v1.2, etc.)

**Tasks**:
1. **Consensus Calculation** (Layer 9)
   - Implement statistical consensus (mean, variance, confidence)
   - Outlier detection (IQR)
   - Owner: Data Scientist
   - Deliverable: consensus_calculator.go (700 lines)

2. **Confidence Scoring** (Layer 9)
   - Compute 0.0-1.0 confidence metric
   - Apply expert weighting
   - Owner: Data Scientist
   - Deliverable: confidence_scorer.go (400 lines)

3. **Version Promotion** (Layer 9)
   - Detect when confidence > 0.85
   - Promote v1.0 → v1.1
   - Owner: Backend Engineer B
   - Deliverable: version_promoter.go (400 lines)

4. **Consensus Evolution Tracking** (Layer 9)
   - Record consensus_evolution entries
   - Blockchain submission for legal admissibility
   - Owner: Data Engineer
   - Deliverable: evolution_tracker.go (300 lines)

5. **Consensus Broadcasting** (Layer 9)
   - Notify all users of new baseline
   - Invalidate cached variants
   - Trigger re-rendering if needed
   - Owner: Network Engineer
   - Deliverable: consensus_broadcaster.go (300 lines)

6. **End-to-End Test** (Layer 9)
   - Feed feedback → compute consensus → promote version
   - Verify all users see v1.1
   - Owner: Integration Engineer
   - Deliverable: consensus_e2e_test.go (400 lines)

**Testing**:
- Consensus calculation: <5 seconds for 10,000 feedback items
- Promotion accuracy: All promotions justified (confidence > 0.85)
- Broadcasting: All clients updated within 1 minute

**Blockers**: None  
**Risks**: Consensus calculation might be slow at scale → parallel processing needed

**Status**: PLANNED

**MILESTONE 3 COMPLETE**: Consensus algorithm live (Weeks 7-9)

---

### Week 10: Multi-Platform Rendering

**Goal**: Render objects on all platforms (Layer 11)

**Tasks**:
1. **Arcade Renderer** (Layer 11)
   - NEO-GEO sprite generation (4-bit color)
   - Rendering hints application
   - ROM compilation
   - Owner: Arcade Engineer
   - Deliverable: arcade_renderer.go (600 lines)

2. **Mobile Renderer** (Layer 11)
   - iOS/Android SVG rendering
   - Touch interaction support
   - Responsive design
   - Owner: Mobile Engineer
   - Deliverable: mobile_renderer.go (500 lines)

3. **Web Renderer** (Layer 11)
   - Browser-based Canvas/SVG rendering
   - Responsive scaling
   - Owner: Frontend Engineer
   - Deliverable: web_renderer.go (400 lines)

4. **UE5 Renderer** (Layer 11)
   - Material instance generation
   - Photorealistic rendering
   - Real-time ray tracing
   - Owner: UE5 Engineer
   - Deliverable: ue5_renderer.go (600 lines)

5. **GIS Renderer** (Layer 11)
   - Cartographic symbol generation (ArcGIS/QGIS)
   - Spatial index usage
   - Owner: GIS Engineer
   - Deliverable: gis_renderer.go (400 lines)

6. **Rendering QA** (Layer 11)
   - Visual regression testing
   - Platform consistency verification
   - Owner: QA Engineer
   - Deliverable: renderer_qa.go (500 lines)

**Testing**:
- All renderers produce valid output
- Cross-platform consistency verified
- Performance targets met (60 FPS arcade, 30 FPS mobile, etc.)

**Blockers**: UE5 engine integration  
**Risks**: Platform-specific nuances → build prototypes for each

**Status**: PLANNED

---

### Week 11: QA & Performance Benchmarking

**Goal**: Comprehensive testing, performance validation (Layer 10)

**Tasks**:
1. **Unit Tests** (Layer 10)
   - 100+ tests covering all functions
   - Target >90% code coverage
   - Owner: QA Engineer
   - Deliverable: *_test.go files (2,000 lines)

2. **Integration Tests** (Layer 10)
   - Multi-layer workflows
   - End-to-end scenarios (sensor → SVG → sync → render)
   - Owner: QA Engineer
   - Deliverable: integration_test.go (800 lines)

3. **Performance Benchmarks** (Layer 10)
   - RTK accuracy: ±5cm
   - Photogrammetry speed: 100 images/60s
   - SVG compression: <50ms, 93% ratio
   - Decompression: <10ms
   - Sync latency: <100ms
   - Consensus: <5s for 10K feedback
   - Rendering: 60 FPS arcade, 30 FPS mobile
   - Owner: Performance Engineer
   - Deliverable: benchmark_suite.go (600 lines)

4. **Stress Testing** (Layer 10)
   - 1,000+ concurrent clients
   - 100,000 objects/sec throughput
   - Memory/CPU monitoring
   - Owner: DevOps Engineer
   - Deliverable: stress_test.go (400 lines)

5. **Security Testing** (Layer 10)
   - HMAC-SHA256 verification
   - TLS encryption validation
   - Auth token verification
   - Owner: Security Engineer
   - Deliverable: security_test.go (300 lines)

**Testing**:
- All tests passing
- Code coverage >90%
- Performance targets met
- No memory leaks
- Security validations pass

**Blockers**: None  
**Risks**: Performance under load → profile and optimize critical paths

**Status**: PLANNED

**MILESTONE 4 COMPLETE**: QA complete, performance validated (Weeks 10-11)

---

### Week 12: Integration, Documentation & Release

**Goal**: Final integration, v0.4.0 release (Layer 13)

**Tasks**:
1. **System Integration** (Layer 13)
   - Wire all 13 layers together
   - End-to-end workflow verification
   - Owner: Integration Lead
   - Deliverable: full_system_test.go (1,000 lines)

2. **Documentation Finalization** (Layer 13)
   - Complete all 10 phase 4.5B documents
   - Update README, API docs
   - Owner: Tech Writer
   - Deliverable: Updated documentation (1,000+ lines)

3. **Release Checklist** (Layer 13)
   - Git tag v0.4.0-universal-save-format
   - Blockchain ledger submission
   - Release notes
   - Owner: Release Manager
   - Deliverable: RELEASE_NOTES_v0.4.0.md

4. **Deployment** (Layer 13)
   - Deploy to staging environment
   - Smoke tests on production-like environment
   - Owner: DevOps Engineer
   - Deliverable: deployment_verified

5. **User Communication** (Layer 13)
   - Announce v0.4.0 to community
   - Migration guide from v0.3.0
   - Owner: Product Manager
   - Deliverable: announcement + migration guide

6. **Archival & Blockchain** (Layer 13)
   - Create immutable blockchain record
   - Archive to IPFS
   - Owner: Data Engineer
   - Deliverable: blockchain transaction confirmed

**Testing**:
- Full system functional test passed
- All documentation complete
- Release artifacts prepared
- Deployment successful

**Blockers**: None (all prior work complete)  
**Risks**: Last-minute integration issues → have rollback plan

**Status**: PLANNED

**FINAL MILESTONE**: v0.4.0 release candidate ready

---

## Team Assignments

```
Total Team: 6-8 engineers

Lead Engineer (1):
  - Integration Lead
  - Final responsibility for v0.4.0 quality
  - Critical path oversight

Backend Engineers (2):
  - Backend Engineer A: Registry, consensus, feedback
  - Backend Engineer B: Recognition, LLM, broadcasting

Computer Vision Engineers (2):
  - CV Engineer A: Photogrammetry (feature detection, matching, SfM)
  - CV Engineer B: Recognition (detection, segmentation, tracking)

Data/DevOps Engineers (1):
  - Data Engineer: SVG extraction, compression, feedback flow
  - Performance/DevOps: Benchmarking, stress testing, deployment

Specialized Engineers (1-2):
  - ML Engineer: LLM integration, model fine-tuning
  - UE5 Engineer: Unreal Engine integration
  - GIS Engineer: Cartographic rendering
  - QA/Test Engineer: Comprehensive testing

Estimated Capacity:
  6 engineers @ 40h/week = 240 hours/week available
  12 weeks = 2,880 hours total
  Estimated effort: ~2,400 hours
  Utilization: ~83% (reasonable with 2 weeks buffer)
```

---

## Risk Mitigation

| Risk | Impact | Mitigation |
|------|--------|-----------|
| RTK initialization slow | 1 week delay | Pre-position base stations, fallback to pseudokinematic |
| MVS library selection | 1 week delay | Build prototypes with 2 options in parallel (Week 2) |
| LLM API rate limits | Ongoing cost | Cache aggressively, batch requests, consider on-premise models |
| UE5 integration complexity | 2 week delay | Start UE5 work Week 1 (parallel track), use plugin template |
| Memory exhaustion on large point clouds | 3 week delay | Stream processing, spatial partitioning, tested on 100M points |
| Consensus calculation slow at scale | 1 week delay | Implement parallel processing, benchmarks in Week 9 |
| Integration failures end-of-sprint | 1 week delay | Daily integration builds, smoke tests on every commit |

---

## Success Criteria

- ✅ All 13 layers implemented and integrated
- ✅ RTK accuracy: ±5cm verified
- ✅ SVG compression: 93% ratio achieved
- ✅ P2P sync: <100ms latency verified
- ✅ Consensus: v1.0 → v1.1 promotion working
- ✅ Multi-platform rendering: All 5 platforms functional
- ✅ Performance: All targets met (60 FPS arcade, etc.)
- ✅ Code coverage: >90% unit test coverage
- ✅ Documentation: All 10 Phase 4.5B documents complete
- ✅ v0.4.0 released to GitHub with blockchain record

---

**Document Status**: ✅ COMPLETE (1,500+ lines)  
**Ready for**: Document 8 (Sensor Integration Guide)

