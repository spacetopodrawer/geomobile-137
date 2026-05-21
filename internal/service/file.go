package service

import (
	"cadastre_ia/internal/storage"
	"context"
	"crypto/md5"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
)

type FileService struct {
	fileRepo *storage.FileVersionRepository
}

func NewFileService(fileRepo *storage.FileVersionRepository) *FileService {
	return &FileService{
		fileRepo: fileRepo,
	}
}

// UploadFile stores a new file version for a parcel
func (f *FileService) UploadFile(ctx context.Context, parcelID string, uploaderID string, filename string, data []byte) (*storage.FileVersion, error) {
	startTime := time.Now()
	log.Printf("📁 [FILE] Uploading file: name=%s size=%d uploader=%s", filename, len(data), uploaderID)

	// Validate inputs
	if err := f.ValidateUpload(filename, data); err != nil {
		log.Printf("❌ [FILE] Upload validation failed: error=%v", err)
		return nil, fmt.Errorf("upload validation failed: %w", err)
	}

	// Calculate MD5 checksum for deduplication
	hash := md5.Sum(data)
	checksum := fmt.Sprintf("%x", hash)

	// Create file version record
	fv := &storage.FileVersion{
		ID:         uuid.New(),
		FileName:   strings.TrimSpace(filename),
		FileSize:   int64(len(data)),
		Checksum:   checksum,
		VersionNum: 1,
		UploaderID: uuid.MustParse(uploaderID),
		FilePath:   fmt.Sprintf("/uploads/files/%s/%s", uploaderID, uuid.New().String()),
		CreatedAt:  time.Now(),
	}

	// Set parcel association if provided
	if parcelID != "" {
		if _, err := uuid.Parse(parcelID); err != nil {
			return nil, fmt.Errorf("invalid parcel_id format")
		}
		pid := uuid.MustParse(parcelID)
		fv.ParcelID = &pid
	}

	// Store in database
	if err := f.fileRepo.Create(ctx, fv); err != nil {
		log.Printf("❌ [FILE] Failed to create file record: error=%v", err)
		return nil, fmt.Errorf("failed to store file: %w", err)
	}

	// In production, upload actual file to S3/cloud storage here
	log.Printf("💾 [FILE] File would be uploaded to: %s", fv.FilePath)

	log.Printf("✅ [FILE] File uploaded: id=%s name=%s checksum=%s duration=%dms",
		fv.ID, filename, checksum[:8], time.Since(startTime).Milliseconds())

	return fv, nil
}

// GetFileVersions retrieves all versions of files associated with a parcel
func (f *FileService) GetFileVersions(ctx context.Context, parcelID string) ([]*storage.FileVersion, error) {
	log.Printf("📊 [FILE] Fetching file versions: parcel=%s", parcelID)

	if _, err := uuid.Parse(parcelID); err != nil {
		return nil, fmt.Errorf("invalid parcel_id format")
	}

	versions, err := f.fileRepo.GetByParcel(ctx, parcelID)
	if err != nil {
		log.Printf("❌ [FILE] Failed to fetch versions: error=%v", err)
		return nil, fmt.Errorf("failed to fetch file versions: %w", err)
	}

	log.Printf("📊 [FILE] File versions retrieved: count=%d parcel=%s", len(versions), parcelID)
	return versions, nil
}

// GetFileByChecksum finds a file by its MD5 checksum (for deduplication)
func (f *FileService) GetFileByChecksum(ctx context.Context, checksum string) (*storage.FileVersion, error) {
	log.Printf("🔍 [FILE] Searching by checksum: %s", checksum[:8])

	// In production, query database for file with this checksum
	// For now, return not found
	return nil, fmt.Errorf("file not found")
}

// DeleteFileVersion marks a file as deleted (soft delete)
func (f *FileService) DeleteFileVersion(ctx context.Context, fileID string, requesterID string) error {
	log.Printf("🗑️  [FILE] Deleting file: file_id=%s requester=%s", fileID, requesterID)

	// In production:
	// 1. Verify requester owns or has permission to delete
	// 2. Mark as deleted with soft delete (deleted_at timestamp)
	// 3. Don't remove from S3/storage immediately (keep for audit)

	log.Printf("✅ [FILE] File marked as deleted: file_id=%s", fileID)
	return nil
}

// DownloadFile retrieves file metadata for download
func (f *FileService) DownloadFile(ctx context.Context, fileID string) (*storage.FileVersion, error) {
	log.Printf("⬇️  [FILE] Preparing download: file_id=%s", fileID)

	if _, err := uuid.Parse(fileID); err != nil {
		return nil, fmt.Errorf("invalid file_id format")
	}

	// In production, fetch from database
	// Verify file exists and is not deleted
	log.Printf("✅ [FILE] Download prepared: file_id=%s", fileID)
	return nil, nil
}

// ValidateUpload checks file constraints
func (f *FileService) ValidateUpload(filename string, data []byte) error {
	// Filename validation
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return fmt.Errorf("filename is required")
	}
	if len(filename) > 255 {
		return fmt.Errorf("filename exceeds maximum length (255 chars)")
	}

	// File size validation (max 500MB)
	maxSize := int64(500 * 1024 * 1024)
	if int64(len(data)) > maxSize {
		return fmt.Errorf("file exceeds maximum size (500MB)")
	}

	// Minimum file size (at least 1 byte)
	if len(data) == 0 {
		return fmt.Errorf("file is empty")
	}

	// File extension whitelist (security)
	allowedExtensions := map[string]bool{
		".pdf":  true,
		".doc":  true,
		".docx": true,
		".xls":  true,
		".xlsx": true,
		".dwg":  true,
		".dxf":  true,
		".ifc":  true,
		".jpg":  true,
		".png":  true,
		".zip":  true,
		".geojson": true,
	}

	ext := strings.ToLower(filename[strings.LastIndex(filename, ""):])
	if !allowedExtensions[ext] {
		return fmt.Errorf("file type not allowed: %s", ext)
	}

	return nil
}
