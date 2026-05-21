package service

import (
	"cadastre_ia/internal/storage"
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
)

type PackageService struct {
	packageRepo *storage.PackageRepository
}

func NewPackageService(packageRepo *storage.PackageRepository) *PackageService {
	return &PackageService{
		packageRepo: packageRepo,
	}
}

// ListPackages retrieves all available feature packages
func (p *PackageService) ListPackages(ctx context.Context) ([]*storage.Package, error) {
	log.Printf("📦 [PKG] Listing all feature packages")

	packages, err := p.packageRepo.GetAll(ctx)
	if err != nil {
		log.Printf("❌ [PKG] Failed to list packages: error=%v", err)
		return nil, fmt.Errorf("failed to list packages: %w", err)
	}

	log.Printf("📦 [PKG] Packages listed: count=%d", len(packages))
	return packages, nil
}

// GetPackage retrieves package metadata by ID
func (p *PackageService) GetPackage(ctx context.Context, id string) (*storage.Package, error) {
	log.Printf("🔍 [PKG] Getting package: id=%s", id)

	if _, err := uuid.Parse(id); err != nil {
		return nil, fmt.Errorf("invalid package_id format")
	}

	pkg, err := p.packageRepo.GetByID(ctx, id)
	if err != nil {
		log.Printf("❌ [PKG] Package not found: error=%v", err)
		return nil, fmt.Errorf("package not found: %w", err)
	}

	log.Printf("✅ [PKG] Package retrieved: id=%s name=%s", pkg.ID, pkg.Name)
	return pkg, nil
}

// CreatePackage registers a new feature package
func (p *PackageService) CreatePackage(ctx context.Context, name, pkgType, version string, size int64, description string, contentPath string, dependencies []string) (*storage.Package, error) {
	startTime := time.Now()
	log.Printf("📥 [PKG] Creating package: name=%s type=%s version=%s", name, pkgType, version)

	// Validate package
	if err := p.ValidatePackage(name, pkgType, version); err != nil {
		log.Printf("❌ [PKG] Validation failed: error=%v", err)
		return nil, fmt.Errorf("package validation failed: %w", err)
	}

	// Create package record
	pkg := &storage.Package{
		ID:           uuid.New(),
		Name:         strings.TrimSpace(name),
		Type:         pkgType,
		Version:      version,
		Size:         size,
		Description:  &description,
		ContentPath:  contentPath,
		Dependencies: dependencies,
		CreatedAt:    time.Now(),
	}

	// Store in database
	if err := p.packageRepo.Create(ctx, pkg); err != nil {
		log.Printf("❌ [PKG] Failed to create package: error=%v", err)
		return nil, fmt.Errorf("failed to create package: %w", err)
	}

	log.Printf("✅ [PKG] Package created: id=%s version=%s size=%d bytes in %dms",
		pkg.ID, version, size, time.Since(startTime).Milliseconds())

	return pkg, nil
}

// GetPackagesByType retrieves packages of a specific type
func (p *PackageService) GetPackagesByType(ctx context.Context, pkgType string) ([]*storage.Package, error) {
	log.Printf("📊 [PKG] Fetching packages by type: %s", pkgType)

	if pkgType = strings.ToLower(strings.TrimSpace(pkgType)); pkgType == "" {
		return nil, fmt.Errorf("package type is required")
	}

	// Validate type
	validTypes := map[string]bool{
		"cad":       true,
		"assets":    true,
		"symbols":   true,
		"animation": true,
	}
	if !validTypes[pkgType] {
		return nil, fmt.Errorf("invalid package type: %s", pkgType)
	}

	packages, err := p.packageRepo.GetByType(ctx, pkgType)
	if err != nil {
		log.Printf("❌ [PKG] Failed to fetch packages: error=%v", err)
		return nil, fmt.Errorf("failed to fetch packages: %w", err)
	}

	log.Printf("📊 [PKG] Packages retrieved: count=%d type=%s", len(packages), pkgType)
	return packages, nil
}

// DownloadPackage initiates package download
func (p *PackageService) DownloadPackage(ctx context.Context, packageID string) (*storage.Package, error) {
	log.Printf("⬇️  [PKG] Downloading package: id=%s", packageID)

	pkg, err := p.GetPackage(ctx, packageID)
	if err != nil {
		return nil, err
	}

	log.Printf("⬇️  [PKG] Download initiated: name=%s size=%d bytes path=%s",
		pkg.Name, pkg.Size, pkg.ContentPath)

	// In production:
	// 1. Check user permissions
	// 2. Log download event
	// 3. Return signed download URL
	// 4. Track bandwidth usage

	return pkg, nil
}

// ValidatePackage ensures package parameters are valid
func (p *PackageService) ValidatePackage(name, pkgType, version string) error {
	// Name validation
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("package name is required")
	}
	if len(name) < 3 || len(name) > 100 {
		return fmt.Errorf("package name must be 3-100 characters")
	}

	// Type validation
	validTypes := map[string]bool{
		"cad":       true,
		"assets":    true,
		"symbols":   true,
		"animation": true,
	}
	pkgType = strings.ToLower(strings.TrimSpace(pkgType))
	if !validTypes[pkgType] {
		return fmt.Errorf("invalid package type: %s", pkgType)
	}

	// Version validation (semantic versioning: X.Y.Z)
	version = strings.TrimSpace(version)
	if version == "" {
		return fmt.Errorf("version is required")
	}

	// Basic semantic version check
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return fmt.Errorf("version must follow semantic versioning (X.Y.Z)")
	}

	return nil
}

// CheckDependencies verifies all package dependencies are available
func (p *PackageService) CheckDependencies(ctx context.Context, dependencies []string) error {
	log.Printf("🔗 [PKG] Checking dependencies: count=%d", len(dependencies))

	for _, depID := range dependencies {
		if _, err := p.GetPackage(ctx, depID); err != nil {
			log.Printf("❌ [PKG] Dependency not found: id=%s", depID)
			return fmt.Errorf("missing dependency: %s", depID)
		}
	}

	log.Printf("✅ [PKG] All dependencies available")
	return nil
}
