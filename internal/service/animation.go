package service

import (
	"cadastre_ia/internal/storage"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
)

type AnimationService struct {
	animRepo *storage.AnimationRepository
}

func NewAnimationService(animRepo *storage.AnimationRepository) *AnimationService {
	return &AnimationService{
		animRepo: animRepo,
	}
}

// CreateAnimation creates a new animation sequence (keyframe or timeseries)
func (a *AnimationService) CreateAnimation(ctx context.Context, name string, animType string, duration float64, keyframes []byte) (*storage.AnimationSequence, error) {
	startTime := time.Now()
	log.Printf("🎬 [ANIM] Creating animation: name=%s type=%s duration=%.2f", name, animType, duration)

	// Validate animation
	if err := a.ValidateAnimation(name, animType, duration); err != nil {
		log.Printf("❌ [ANIM] Validation failed: error=%v", err)
		return nil, fmt.Errorf("animation validation failed: %w", err)
	}

	// Create animation record
	anim := &storage.AnimationSequence{
		ID:        uuid.New(),
		Name:      strings.TrimSpace(name),
		Type:      animType,
		Duration:  duration,
		Keyframes: keyframes,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Store in database
	if err := a.animRepo.Create(ctx, anim); err != nil {
		log.Printf("❌ [ANIM] Failed to create animation: error=%v", err)
		return nil, fmt.Errorf("failed to create animation: %w", err)
	}

	log.Printf("✅ [ANIM] Animation created: id=%s name=%s duration=%.2fs in %dms",
		anim.ID, name, duration, time.Since(startTime).Milliseconds())

	return anim, nil
}

// GetAnimation retrieves animation metadata
func (a *AnimationService) GetAnimation(ctx context.Context, id string) (*storage.AnimationSequence, error) {
	log.Printf("🔍 [ANIM] Getting animation: id=%s", id)

	if _, err := uuid.Parse(id); err != nil {
		return nil, fmt.Errorf("invalid animation_id format")
	}

	anim, err := a.animRepo.GetByID(ctx, id)
	if err != nil {
		log.Printf("❌ [ANIM] Animation not found: error=%v", err)
		return nil, fmt.Errorf("animation not found: %w", err)
	}

	log.Printf("✅ [ANIM] Animation retrieved: id=%s name=%s", anim.ID, anim.Name)
	return anim, nil
}

// UpdateAnimation modifies animation properties
func (a *AnimationService) UpdateAnimation(ctx context.Context, id string, name string, duration float64, keyframes []byte) (*storage.AnimationSequence, error) {
	startTime := time.Now()
	log.Printf("✏️  [ANIM] Updating animation: id=%s", id)

	// Fetch existing animation
	anim, err := a.GetAnimation(ctx, id)
	if err != nil {
		return nil, err
	}

	// Validate updates
	if err := a.ValidateAnimation(name, anim.Type, duration); err != nil {
		return nil, fmt.Errorf("update validation failed: %w", err)
	}

	// Update fields
	anim.Name = strings.TrimSpace(name)
	anim.Duration = duration
	if keyframes != nil {
		anim.Keyframes = keyframes
	}
	anim.UpdatedAt = time.Now()

	// Persist changes
	if err := a.animRepo.Update(ctx, anim); err != nil {
		log.Printf("❌ [ANIM] Failed to update animation: error=%v", err)
		return nil, fmt.Errorf("failed to update animation: %w", err)
	}

	log.Printf("✅ [ANIM] Animation updated: id=%s duration=%.2fs in %dms",
		id, duration, time.Since(startTime).Milliseconds())

	return anim, nil
}

// PlayAnimation initiates animation playback (broadcasts to clients)
func (a *AnimationService) PlayAnimation(ctx context.Context, animID string, loop bool) error {
	log.Printf("▶️  [ANIM] Playing animation: id=%s loop=%v", animID, loop)

	// Fetch animation to verify exists
	anim, err := a.GetAnimation(ctx, animID)
	if err != nil {
		return err
	}

	log.Printf("▶️  [ANIM] Animation playback started: name=%s duration=%.2fs", anim.Name, anim.Duration)
	// In production: broadcast play event to WebSocket clients
	return nil
}

// StopAnimation halts animation playback
func (a *AnimationService) StopAnimation(ctx context.Context, animID string) error {
	log.Printf("⏹️  [ANIM] Stopping animation: id=%s", animID)

	if _, err := uuid.Parse(animID); err != nil {
		return fmt.Errorf("invalid animation_id")
	}

	log.Printf("⏹️  [ANIM] Animation stopped: id=%s", animID)
	// In production: broadcast stop event to WebSocket clients
	return nil
}

// ValidateAnimation ensures animation parameters are valid
func (a *AnimationService) ValidateAnimation(name, animType string, duration float64) error {
	// Name validation
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("animation name is required")
	}
	if len(name) > 100 {
		return fmt.Errorf("animation name exceeds max length (100 chars)")
	}

	// Type validation
	validTypes := map[string]bool{
		"keyframe":   true,
		"timeseries": true,
		"property":   true,
	}
	animType = strings.ToLower(strings.TrimSpace(animType))
	if !validTypes[animType] {
		return fmt.Errorf("invalid animation type: %s", animType)
	}

	// Duration validation
	if duration <= 0 {
		return fmt.Errorf("duration must be greater than 0")
	}
	if duration > 3600 { // Max 1 hour
		return fmt.Errorf("duration exceeds maximum (3600 seconds)")
	}

	return nil
}

// ValidateKeyframes ensures keyframe data is valid JSON
func (a *AnimationService) ValidateKeyframes(keyframesJSON []byte) error {
	var keyframes []map[string]interface{}
	if err := json.Unmarshal(keyframesJSON, &keyframes); err != nil {
		return fmt.Errorf("invalid keyframes JSON: %w", err)
	}

	if len(keyframes) == 0 {
		return fmt.Errorf("at least one keyframe is required")
	}

	return nil
}
