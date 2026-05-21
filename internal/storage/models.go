package storage

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Name         string    `json:"name"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Workspace struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	OwnerID   uuid.UUID `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Parcel struct {
	ID          uuid.UUID       `json:"id"`
	WorkspaceID uuid.UUID       `json:"workspace_id"`
	ParcelNum   string          `json:"parcel_number"`
	Geometry    string          `json:"geometry"`
	OwnerID     *uuid.UUID      `json:"owner_id,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	Version     int             `json:"version"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type SyncOperation struct {
	ID            int64           `json:"id"`
	ParcelID      uuid.UUID       `json:"parcel_id"`
	ClientID      uuid.UUID       `json:"client_id"`
	OperationType string          `json:"operation_type"`
	OperationData json.RawMessage `json:"operation_data"`
	Timestamp     int64           `json:"timestamp"`
	AppliedAt     time.Time       `json:"applied_at"`
	Status        string          `json:"status"`
}

type Conflict struct {
	ID              uuid.UUID       `json:"id"`
	ParcelID        uuid.UUID       `json:"parcel_id"`
	Operation1ID    *int64          `json:"operation1_id,omitempty"`
	Operation2ID    *int64          `json:"operation2_id,omitempty"`
	ResolutionType  *string         `json:"resolution_type,omitempty"`
	ResolvedData    json.RawMessage `json:"resolved_data,omitempty"`
	ResolvedAt      time.Time       `json:"resolved_at"`
}

type CADFile struct {
	ID                 uuid.UUID `json:"id"`
	ParcelID           *uuid.UUID `json:"parcel_id,omitempty"`
	UploaderID         uuid.UUID `json:"uploader_id"`
	FileFormat         string    `json:"file_format"`
	FilePath           string    `json:"file_path"`
	OriginalFilename   string    `json:"original_filename"`
	ConvertedGeometry  *string   `json:"converted_geometry,omitempty"`
	ConversionStatus   string    `json:"conversion_status"`
	ConversionError    *string   `json:"conversion_error,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type Asset3D struct {
	ID             uuid.UUID       `json:"id"`
	PackID         *uuid.UUID      `json:"pack_id,omitempty"`
	Name           string          `json:"name"`
	Category       string          `json:"category"`
	ModelURL       string          `json:"model_url"`
	ModelFormat    *string         `json:"model_format,omitempty"`
	Scale          float64         `json:"scale"`
	PreviewIcon    *string         `json:"preview_icon,omitempty"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}

type AssetPlacement struct {
	ID        uuid.UUID `json:"id"`
	ParcelID  uuid.UUID `json:"parcel_id"`
	AssetID   uuid.UUID `json:"asset_id"`
	Position  string    `json:"position"`
	Rotation  float64   `json:"rotation"`
	Scale     float64   `json:"scale"`
	PlacedBy  uuid.UUID `json:"placed_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Message struct {
	ID        uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	SenderID  uuid.UUID `json:"sender_id"`
	Content   string    `json:"content"`
	ThreadID  *uuid.UUID `json:"thread_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	EditedAt  *time.Time `json:"edited_at,omitempty"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

type FileVersion struct {
	ID         uuid.UUID `json:"id"`
	ParcelID   *uuid.UUID `json:"parcel_id,omitempty"`
	UploaderID uuid.UUID `json:"uploader_id"`
	FileName   string    `json:"file_name"`
	FileSize   int64     `json:"file_size"`
	FilePath   string    `json:"file_path"`
	Checksum   string    `json:"checksum"`
	VersionNum int       `json:"version_number"`
	CreatedAt  time.Time `json:"created_at"`
}

type AnimationSequence struct {
	ID                uuid.UUID       `json:"id"`
	ParcelID          *uuid.UUID      `json:"parcel_id,omitempty"`
	AssetPlacementID  *uuid.UUID      `json:"asset_placement_id,omitempty"`
	Type              string          `json:"type"`
	Name              string          `json:"name"`
	Duration          float64         `json:"duration"`
	Keyframes         json.RawMessage `json:"keyframes,omitempty"`
	TimeseriesData    json.RawMessage `json:"timeseries_data,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type Package struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	Version      string    `json:"version"`
	Size         int64     `json:"size"`
	Description  *string   `json:"description,omitempty"`
	ContentPath  string    `json:"content_path"`
	Dependencies []string  `json:"dependencies,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type UserSession struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	TokenHash string    `json:"token_hash"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type AuditLog struct {
	ID           int64           `json:"id"`
	UserID       *uuid.UUID      `json:"user_id,omitempty"`
	Action       string          `json:"action"`
	ResourceType *string         `json:"resource_type,omitempty"`
	ResourceID   *uuid.UUID      `json:"resource_id,omitempty"`
	Changes      json.RawMessage `json:"changes,omitempty"`
	IPAddress    *string         `json:"ip_address,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

// JSONData is a wrapper type for json.RawMessage that implements driver.Valuer
type JSONData json.RawMessage

func (jd JSONData) Value() (driver.Value, error) {
	if len(jd) == 0 {
		return nil, nil
	}
	return []byte(jd), nil
}

// For compatibility, keep the function but remove the problematic method receiver
// MarshalJSON returns the JSON encoding of JSONData
func (jd JSONData) MarshalJSON() ([]byte, error) {
	if len(jd) == 0 {
		return []byte("null"), nil
	}
	return []byte(jd), nil
}

func MustParseUUID(value string) uuid.UUID {
	return uuid.MustParse(value)
}
