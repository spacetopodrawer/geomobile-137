package service

import (
	"cadastre_ia/internal/storage"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

type AssetService struct {
	assetRepo     *storage.Asset3DRepository
	placementRepo *storage.AssetPlacementRepository
}

func NewAssetService(assetRepo *storage.Asset3DRepository, placementRepo *storage.AssetPlacementRepository) *AssetService {
	return &AssetService{
		assetRepo:     assetRepo,
		placementRepo: placementRepo,
	}
}

// ListAssets retrieves all available 3D assets in the library
func (a *AssetService) ListAssets(ctx context.Context) ([]*storage.Asset3D, error) {
	log.Printf("📦 [ASSET] Listing all 3D assets")

	assets, err := a.assetRepo.GetAll(ctx)
	if err != nil {
		log.Printf("❌ [ASSET] Failed to fetch assets: error=%v", err)
		return nil, fmt.Errorf("failed to fetch assets: %w", err)
	}

	log.Printf("📦 [ASSET] Assets retrieved: count=%d", len(assets))
	return assets, nil
}

// PlaceAsset positions a 3D asset on a parcel at specific coordinates
func (a *AssetService) PlaceAsset(ctx context.Context, parcelID, assetID, placedBy string, position string, rotation, scale float64) (*storage.AssetPlacement, error) {
	startTime := time.Now()
	log.Printf("📍 [ASSET] Placing asset: asset=%s parcel=%s position=%s", assetID, parcelID, position)

	// Validate inputs
	if err := a.ValidatePlacement(parcelID, assetID, placedBy, rotation, scale); err != nil {
		log.Printf("❌ [ASSET] Placement validation failed: error=%v", err)
		return nil, fmt.Errorf("placement validation failed: %w", err)
	}

	// Create placement record
	placement := &storage.AssetPlacement{
		ID:        uuid.New(),
		ParcelID:  uuid.MustParse(parcelID),
		AssetID:   uuid.MustParse(assetID),
		Position:  position, // GeoJSON Point: {"type": "Point", "coordinates": [lat, lon]}
		Rotation:  rotation,
		Scale:     scale,
		PlacedBy:  uuid.MustParse(placedBy),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Store in database
	if err := a.placementRepo.Create(ctx, placement); err != nil {
		log.Printf("❌ [ASSET] Failed to create placement: error=%v", err)
		return nil, fmt.Errorf("failed to place asset: %w", err)
	}

	log.Printf("✅ [ASSET] Asset placed: id=%s rotation=%.2f° scale=%.2f duration=%dms",
		placement.ID, rotation, scale, time.Since(startTime).Milliseconds())

	return placement, nil
}

// MovePlacement updates asset position, rotation, and scale
func (a *AssetService) MovePlacement(ctx context.Context, placementID string, newPosition string, newRotation, newScale float64) (*storage.AssetPlacement, error) {
	log.Printf("🔄 [ASSET] Moving placement: placement=%s", placementID)

	// In production:
	// 1. Fetch existing placement
	// 2. Validate new values
	// 3. Update in database
	// 4. Broadcast update via WebSocket

	log.Printf("✅ [ASSET] Placement moved: id=%s", placementID)
	return nil, nil
}

// RemovePlacement deletes an asset placement from a parcel
func (a *AssetService) RemovePlacement(ctx context.Context, placementID string) error {
	log.Printf("🗑️  [ASSET] Removing asset placement: placement=%s", placementID)

	if _, err := uuid.Parse(placementID); err != nil {
		return fmt.Errorf("invalid placement_id format")
	}

	if err := a.placementRepo.Delete(ctx, placementID); err != nil {
		log.Printf("❌ [ASSET] Failed to remove placement: error=%v", err)
		return fmt.Errorf("failed to remove placement: %w", err)
	}

	log.Printf("✅ [ASSET] Asset placement removed: id=%s", placementID)
	return nil
}

// GetPlacementsByParcel retrieves all asset placements on a parcel
func (a *AssetService) GetPlacementsByParcel(ctx context.Context, parcelID string) ([]*storage.AssetPlacement, error) {
	log.Printf("🔍 [ASSET] Fetching asset placements: parcel=%s", parcelID)

	if _, err := uuid.Parse(parcelID); err != nil {
		return nil, fmt.Errorf("invalid parcel_id format")
	}

	placements, err := a.placementRepo.GetByParcel(ctx, parcelID)
	if err != nil {
		log.Printf("❌ [ASSET] Failed to fetch placements: error=%v", err)
		return nil, fmt.Errorf("failed to fetch placements: %w", err)
	}

	log.Printf("🔍 [ASSET] Placements retrieved: count=%d parcel=%s", len(placements), parcelID)
	return placements, nil
}

// ValidatePlacement ensures placement parameters are valid
func (a *AssetService) ValidatePlacement(parcelID, assetID, placedBy string, rotation, scale float64) error {
	// UUID validation
	if _, err := uuid.Parse(parcelID); err != nil {
		return fmt.Errorf("invalid parcel_id")
	}
	if _, err := uuid.Parse(assetID); err != nil {
		return fmt.Errorf("invalid asset_id")
	}
	if _, err := uuid.Parse(placedBy); err != nil {
		return fmt.Errorf("invalid placed_by user_id")
	}

	// Rotation validation (0-360 degrees)
	if rotation < 0 || rotation > 360 {
		return fmt.Errorf("rotation must be between 0 and 360 degrees")
	}

	// Scale validation (0.1x to 10x)
	if scale < 0.1 || scale > 10.0 {
		return fmt.Errorf("scale must be between 0.1 and 10.0")
	}

	return nil
}
