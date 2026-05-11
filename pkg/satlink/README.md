# 🛰️ Module pkg/satlink — Satellite Decoder & TV Integration

**Version:** 0.1.0 (Design)  
**Status:** R&D Phase  
**Part of:** geo-mobile137 v0.3.0+ (Planned)

---

## 📌 Overview

The `satlink` module provides integration with **satellite data providers** (SES, Eutelsat, Bell Sat) for real-time positioning, imagery, and TV channel overlay on cadastral maps.

### Three-Tier Decoder System

| Tier | Name | Monthly Price | Features | Target Users |
|------|------|---------------|----------|--------------|
| **Free** | Educational | Free (sponsored) | Public satellite signals, OSM maps, educational content | Students, NGOs |
| **RTK** | Professional | 29,900 XAF | Centimeter-level precision (RTK), advanced filtering | Surveyors, land agents |
| **Premium** | Enterprise | 59,900 XAF | RTK + TV integration + Lékié Quest VR unlock + API | Municipalities, enterprises |

---

## 🎯 Components

### 1. Satellite Signal Decoder

```go
// pkg/satlink/decoder.go

type SignalDecoder struct {
  FrequencyGHz    float64       // e.g., 11.475 GHz for Hotbird
  Polarization    string        // H or V (horizontal/vertical)
  SymbolRateMbps  float32       // 27.5 Mbps typical
  FECRate         string        // 5/6, 3/4, 7/8
  SignalQuality   float64       // 0-100% (SNR indicator)
  AvailableNow    bool
  NextWindowAt    *time.Time    // For geostationary satellites
}

// Decode MPEG-TS stream from satellite
func (d *SignalDecoder) DecodeMPEGTS(signal []byte) (*TSPacket, error) {
  // Parse MPEG-TS packets (188 bytes each)
  // Extract CAM (Conditional Access) module if encrypted
  // Decrypt if user has valid CAM
}
```

### 2. RTK (Real-Time Kinematic) Positioning

```go
// Real-time centimeter-level precision
type RTKProvider struct {
  SatelliteConstellation string  // "GPS", "GLONASS", "Galileo", "BeiDou"
  BaseStationCoords     Point    // Reference ground station
  RoverCoords           Point    // Mobile device
  Accuracy              float64  // ±2cm typical
  UpdateRateHz          int      // 10Hz typical
  SessionID             string
}

// Methods:
// - InitRTKSession(baseCoords) → sessionID
// - UpdateRoverPosition(rawSignal) → Point (cm-level accuracy)
// - GetDilutionOfPrecision() → float64 (GDOP value)
```

### 3. TV Channel Overlay

```go
// pkg/satlink/tv_overlay.go

type TVChannel struct {
  ID            string
  Name          string  // "Cameroon National TV", "RTCinfo"
  Frequency     float64 // GHz
  IsEncrypted   bool
  DataStandard  string  // "DVB-S2", "DVB-C", "ISDB-T"
  LogoURL       string
}

type MapOverlay struct {
  Channel       *TVChannel
  ContentType   string  // "weather", "news", "sports", "cadastre"
  SVGContent    string  // SVG XML for map overlay
  UpdateFrequency time.Duration
  Geolocation   *BoundingBox  // Valid region for this overlay
}

// Example: SatLink broadcasts real-time cadastral updates on TV
// "Tune to RTCinfo + engage Lékié Quest VR" = two-screen experience
```

---

## 💳 Monetization Model

### Revenue Projections (P1)

- **Phase 1 (Cameroon):** 0 FCFA (R&D only)
- **Phase 2 (CEMAC):** 10-20M FCFA/year (5-10 RTK subscribers)
- **Phase 3+ (Africa):** 50-500M FCFA/year (50-500 RTK + 100K Free users)

### Licensing Strategy

**RTK/Premium tiers require:**
1. **Operator License** from ART (Cameroon telecom regulator) or equivalent
2. **Satellite Transponder Lease** (SES/Eutelsat, ~$500K-1M USD/year for shared capacity)
3. **TV Broadcast Rights** (if TV overlay feature used)

**Note:** SatLink is intentionally **expensive to license**, so P1 focus is **R&D partnerships** with government bodies.

---

## 🏗️ Integration with Cadastre_IA

### Use Case: Surveyors in the Field

```
Surveyor in field (Monatélé, Lékié):
  1. Opens Cadastre_IA Pro on mobile
  2. Activates SatLink RTK tier (pay 29,900 FCFA)
  3. Receives centimeter-level GPS fix (vs ±5m with standard GPS)
  4. Maps parcel boundaries with ±2cm accuracy
  5. Disputes resolved faster (no longer "did you measure right?")
```

### Technical Integration

```
Cadastre_IA API
  ├─ /api/v1/satlink/rtk/enable      (activate RTK, check subscription)
  ├─ /api/v1/satlink/position        (get current position, cm-level)
  ├─ /api/v1/satlink/tv/channels     (list available TV channels)
  └─ /api/v1/satlink/tv/overlay      (fetch cadastral overlay for channel)

SatLink Module
  ├─ Signal Decoder (raw satellite data → usable signal)
  ├─ RTK Engine (GNSS corrections → position)
  └─ TV Broadcast Manager (schedule + distribute cadastral updates)
```

---

## 🔐 Security & Compliance

### CAM (Conditional Access) Module

- Encrypted channels require CAM card (smartcard or software)
- **Enterprise version:** Hardware CAM support for institutional deployments
- **Compliance:** Follows DVB-CSA encryption standard

### Data Governance

- **Real-time cadastral updates on TV:** Only public information (no owner names, no disputes)
- **RTK positioning data:** Encrypted end-to-end (surveyor ↔ server)
- **Audit logs:** All position fixes logged (for compliance verification)

---

## 📊 Implementation Roadmap

### P1 (M0-12): Foundation
- [ ] R&D partnership with SES or Eutelsat
- [ ] Proof-of-concept signal decoder (mockup)
- [ ] RTK engine prototype (using open-source RTKLIB)
- [ ] Licensing agreements drafted

### P2 (M12-24): Deployment
- [ ] RTK service live (Cameroon + CEMAC)
- [ ] 5-10 paying RTK subscribers
- [ ] TV broadcast integration (1-2 trial channels)

### P3 (M24-36): Expansion
- [ ] 50+ RTK subscribers across Africa
- [ ] 100K+ Free tier users
- [ ] Regional TV partnerships (10+ channels)

### P4 (M36+): Scale
- [ ] Global SatLink coverage
- [ ] 500+ RTK subscribers
- [ ] 1M+ Free users
- [ ] Recurring revenue: 50-100M FCFA/month

---

## 🧪 Testing Strategy

```bash
# Signal quality simulation
go test ./pkg/satlink/decoder_test.go -v

# RTK positioning accuracy
go test ./pkg/satlink/rtk_test.go -v

# TV overlay rendering
go test ./pkg/satlink/tv_overlay_test.go -v

# Load test (1000 concurrent RTK users)
go run loadtest.go --rtk-users=1000
```

---

## 📚 References

- DVB Standards: https://www.dvb.org/
- RTKLIB (open-source RTK): https://github.com/tomojitakasu/RTKLIB
- SES Satellite Services: https://www.ses.com/
- Eutelsat: https://www.eutelsat.com/

---

**Status:** Design Phase (not yet coded)  
**Next:** Finalize licensing agreements before development begins  
**Assigned to:** Wilfried (R&D lead) + Cowork (technical specification)

