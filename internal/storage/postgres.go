package storage

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Storage struct {
	db *pgxpool.Pool
}

func NewStorage(pool *pgxpool.Pool) *Storage {
	return &Storage{db: pool}
}

func (s *Storage) GetDB() *pgxpool.Pool {
	return s.db
}

func (s *Storage) Close() {
	s.db.Close()
}

type ParcelRepository struct {
	db *pgxpool.Pool
}

func NewParcelRepository(pool *pgxpool.Pool) *ParcelRepository {
	return &ParcelRepository{db: pool}
}

func (r *ParcelRepository) GetByID(ctx context.Context, id string) (*Parcel, error) {
	var p Parcel
	err := r.db.QueryRow(ctx,
		"SELECT id, workspace_id, parcel_number, ST_AsText(geometry), owner_id, metadata, version, created_at, updated_at FROM parcels WHERE id = $1",
		id,
	).Scan(&p.ID, &p.WorkspaceID, &p.ParcelNum, &p.Geometry, &p.OwnerID, &p.Metadata, &p.Version, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ParcelRepository) GetByWorkspace(ctx context.Context, workspaceID string) ([]*Parcel, error) {
	rows, err := r.db.Query(ctx,
		"SELECT id, workspace_id, parcel_number, ST_AsText(geometry), owner_id, metadata, version, created_at, updated_at FROM parcels WHERE workspace_id = $1",
		workspaceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var parcels []*Parcel
	for rows.Next() {
		var p Parcel
		if err := rows.Scan(&p.ID, &p.WorkspaceID, &p.ParcelNum, &p.Geometry, &p.OwnerID, &p.Metadata, &p.Version, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		parcels = append(parcels, &p)
	}
	return parcels, rows.Err()
}

func (r *ParcelRepository) Create(ctx context.Context, p *Parcel) error {
	err := r.db.QueryRow(ctx,
		"INSERT INTO parcels (workspace_id, parcel_number, geometry, owner_id, metadata, version) VALUES ($1, $2, ST_GeomFromText($3, 4326), $4, $5, $6) RETURNING id, created_at, updated_at",
		p.WorkspaceID, p.ParcelNum, p.Geometry, p.OwnerID, p.Metadata, p.Version,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	return err
}

func (r *ParcelRepository) Update(ctx context.Context, p *Parcel) error {
	_, err := r.db.Exec(ctx,
		"UPDATE parcels SET parcel_number = $1, geometry = ST_GeomFromText($2, 4326), owner_id = $3, metadata = $4, version = $5, updated_at = NOW() WHERE id = $6",
		p.ParcelNum, p.Geometry, p.OwnerID, p.Metadata, p.Version, p.ID,
	)
	return err
}

func (r *ParcelRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, "DELETE FROM parcels WHERE id = $1", id)
	return err
}

type SyncOperationRepository struct {
	db *pgxpool.Pool
}

func NewSyncOperationRepository(pool *pgxpool.Pool) *SyncOperationRepository {
	return &SyncOperationRepository{db: pool}
}

func (r *SyncOperationRepository) GetByParcel(ctx context.Context, parcelID string) ([]*SyncOperation, error) {
	rows, err := r.db.Query(ctx,
		"SELECT id, parcel_id, client_id, operation_type, operation_data, timestamp, applied_at, status FROM sync_operations WHERE parcel_id = $1 ORDER BY timestamp ASC",
		parcelID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ops []*SyncOperation
	for rows.Next() {
		var op SyncOperation
		if err := rows.Scan(&op.ID, &op.ParcelID, &op.ClientID, &op.OperationType, &op.OperationData, &op.Timestamp, &op.AppliedAt, &op.Status); err != nil {
			return nil, err
		}
		ops = append(ops, &op)
	}
	return ops, rows.Err()
}

func (r *SyncOperationRepository) Create(ctx context.Context, op *SyncOperation) error {
	err := r.db.QueryRow(ctx,
		"INSERT INTO sync_operations (parcel_id, client_id, operation_type, operation_data, timestamp, status) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, applied_at",
		op.ParcelID, op.ClientID, op.OperationType, op.OperationData, op.Timestamp, op.Status,
	).Scan(&op.ID, &op.AppliedAt)
	return err
}

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: pool}
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := r.db.QueryRow(ctx,
		"SELECT id, email, password_hash, name, created_at, updated_at FROM users WHERE email = $1",
		email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*User, error) {
	var u User
	err := r.db.QueryRow(ctx,
		"SELECT id, email, password_hash, name, created_at, updated_at FROM users WHERE id = $1",
		id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) Create(ctx context.Context, u *User) error {
	err := r.db.QueryRow(ctx,
		"INSERT INTO users (email, password_hash, name) VALUES ($1, $2, $3) RETURNING id, created_at, updated_at",
		u.Email, u.PasswordHash, u.Name,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
	return err
}

type CADFileRepository struct {
	db *pgxpool.Pool
}

func NewCADFileRepository(pool *pgxpool.Pool) *CADFileRepository {
	return &CADFileRepository{db: pool}
}

func (r *CADFileRepository) GetByID(ctx context.Context, id string) (*CADFile, error) {
	var c CADFile
	err := r.db.QueryRow(ctx,
		"SELECT id, parcel_id, uploader_id, file_format, file_path, original_filename, converted_geometry, conversion_status, conversion_error, created_at, updated_at FROM cad_files WHERE id = $1",
		id,
	).Scan(&c.ID, &c.ParcelID, &c.UploaderID, &c.FileFormat, &c.FilePath, &c.OriginalFilename, &c.ConvertedGeometry, &c.ConversionStatus, &c.ConversionError, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CADFileRepository) Create(ctx context.Context, c *CADFile) error {
	err := r.db.QueryRow(ctx,
		"INSERT INTO cad_files (parcel_id, uploader_id, file_format, file_path, original_filename, conversion_status) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at, updated_at",
		c.ParcelID, c.UploaderID, c.FileFormat, c.FilePath, c.OriginalFilename, c.ConversionStatus,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
	return err
}

func (r *CADFileRepository) Update(ctx context.Context, c *CADFile) error {
	_, err := r.db.Exec(ctx,
		"UPDATE cad_files SET conversion_status = $1, converted_geometry = $2, conversion_error = $3, updated_at = NOW() WHERE id = $4",
		c.ConversionStatus, c.ConvertedGeometry, c.ConversionError, c.ID,
	)
	return err
}

type MessageRepository struct {
	db *pgxpool.Pool
}

func NewMessageRepository(pool *pgxpool.Pool) *MessageRepository {
	return &MessageRepository{db: pool}
}

func (r *MessageRepository) GetByWorkspace(ctx context.Context, workspaceID string) ([]*Message, error) {
	rows, err := r.db.Query(ctx,
		"SELECT id, workspace_id, sender_id, content, thread_id, created_at, edited_at, deleted_at FROM messages WHERE workspace_id = $1 ORDER BY created_at DESC",
		workspaceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.WorkspaceID, &m.SenderID, &m.Content, &m.ThreadID, &m.CreatedAt, &m.EditedAt, &m.DeletedAt); err != nil {
			return nil, err
		}
		messages = append(messages, &m)
	}
	return messages, rows.Err()
}

func (r *MessageRepository) GetByThread(ctx context.Context, threadID string) ([]*Message, error) {
	rows, err := r.db.Query(ctx,
		"SELECT id, workspace_id, sender_id, content, thread_id, created_at, edited_at, deleted_at FROM messages WHERE thread_id = $1 ORDER BY created_at ASC",
		threadID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.WorkspaceID, &m.SenderID, &m.Content, &m.ThreadID, &m.CreatedAt, &m.EditedAt, &m.DeletedAt); err != nil {
			return nil, err
		}
		messages = append(messages, &m)
	}
	return messages, rows.Err()
}

func (r *MessageRepository) Create(ctx context.Context, m *Message) error {
	err := r.db.QueryRow(ctx,
		"INSERT INTO messages (workspace_id, sender_id, content, thread_id) VALUES ($1, $2, $3, $4) RETURNING id, created_at",
		m.WorkspaceID, m.SenderID, m.Content, m.ThreadID,
	).Scan(&m.ID, &m.CreatedAt)
	return err
}

type FileVersionRepository struct {
	db *pgxpool.Pool
}

func NewFileVersionRepository(pool *pgxpool.Pool) *FileVersionRepository {
	return &FileVersionRepository{db: pool}
}

func (r *FileVersionRepository) GetByParcel(ctx context.Context, parcelID string) ([]*FileVersion, error) {
	rows, err := r.db.Query(ctx,
		"SELECT id, parcel_id, uploader_id, file_name, file_size, file_path, checksum, version_number, created_at FROM file_versions WHERE parcel_id = $1 ORDER BY created_at DESC",
		parcelID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*FileVersion
	for rows.Next() {
		var fv FileVersion
		if err := rows.Scan(&fv.ID, &fv.ParcelID, &fv.UploaderID, &fv.FileName, &fv.FileSize, &fv.FilePath, &fv.Checksum, &fv.VersionNum, &fv.CreatedAt); err != nil {
			return nil, err
		}
		files = append(files, &fv)
	}
	return files, rows.Err()
}

func (r *FileVersionRepository) Create(ctx context.Context, fv *FileVersion) error {
	err := r.db.QueryRow(ctx,
		"INSERT INTO file_versions (parcel_id, uploader_id, file_name, file_size, file_path, checksum, version_number) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, created_at",
		fv.ParcelID, fv.UploaderID, fv.FileName, fv.FileSize, fv.FilePath, fv.Checksum, fv.VersionNum,
	).Scan(&fv.ID, &fv.CreatedAt)
	return err
}

type AnimationRepository struct {
	db *pgxpool.Pool
}

func NewAnimationRepository(pool *pgxpool.Pool) *AnimationRepository {
	return &AnimationRepository{db: pool}
}

func (r *AnimationRepository) GetByID(ctx context.Context, id string) (*AnimationSequence, error) {
	var a AnimationSequence
	err := r.db.QueryRow(ctx,
		"SELECT id, parcel_id, asset_placement_id, type, name, duration, keyframes, timeseries_data, created_at, updated_at FROM animation_sequences WHERE id = $1",
		id,
	).Scan(&a.ID, &a.ParcelID, &a.AssetPlacementID, &a.Type, &a.Name, &a.Duration, &a.Keyframes, &a.TimeseriesData, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *AnimationRepository) Create(ctx context.Context, a *AnimationSequence) error {
	err := r.db.QueryRow(ctx,
		"INSERT INTO animation_sequences (parcel_id, asset_placement_id, type, name, duration, keyframes, timeseries_data) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, created_at, updated_at",
		a.ParcelID, a.AssetPlacementID, a.Type, a.Name, a.Duration, a.Keyframes, a.TimeseriesData,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
	return err
}

func (r *AnimationRepository) Update(ctx context.Context, a *AnimationSequence) error {
	_, err := r.db.Exec(ctx,
		"UPDATE animation_sequences SET name = $1, duration = $2, keyframes = $3, timeseries_data = $4, updated_at = NOW() WHERE id = $5",
		a.Name, a.Duration, a.Keyframes, a.TimeseriesData, a.ID,
	)
	return err
}

type Asset3DRepository struct {
	db *pgxpool.Pool
}

func NewAsset3DRepository(pool *pgxpool.Pool) *Asset3DRepository {
	return &Asset3DRepository{db: pool}
}

func (r *Asset3DRepository) GetAll(ctx context.Context) ([]*Asset3D, error) {
	rows, err := r.db.Query(ctx,
		"SELECT id, pack_id, name, category, model_url, model_format, scale, preview_icon, metadata, created_at FROM assets_3d ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []*Asset3D
	for rows.Next() {
		var a Asset3D
		if err := rows.Scan(&a.ID, &a.PackID, &a.Name, &a.Category, &a.ModelURL, &a.ModelFormat, &a.Scale, &a.PreviewIcon, &a.Metadata, &a.CreatedAt); err != nil {
			return nil, err
		}
		assets = append(assets, &a)
	}
	return assets, rows.Err()
}

type AssetPlacementRepository struct {
	db *pgxpool.Pool
}

func NewAssetPlacementRepository(pool *pgxpool.Pool) *AssetPlacementRepository {
	return &AssetPlacementRepository{db: pool}
}

func (r *AssetPlacementRepository) GetByParcel(ctx context.Context, parcelID string) ([]*AssetPlacement, error) {
	rows, err := r.db.Query(ctx,
		"SELECT id, parcel_id, asset_id, position, rotation, scale, placed_by, created_at, updated_at FROM asset_placements WHERE parcel_id = $1",
		parcelID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var placements []*AssetPlacement
	for rows.Next() {
		var ap AssetPlacement
		if err := rows.Scan(&ap.ID, &ap.ParcelID, &ap.AssetID, &ap.Position, &ap.Rotation, &ap.Scale, &ap.PlacedBy, &ap.CreatedAt, &ap.UpdatedAt); err != nil {
			return nil, err
		}
		placements = append(placements, &ap)
	}
	return placements, rows.Err()
}

func (r *AssetPlacementRepository) Create(ctx context.Context, ap *AssetPlacement) error {
	err := r.db.QueryRow(ctx,
		"INSERT INTO asset_placements (parcel_id, asset_id, position, rotation, scale, placed_by) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at, updated_at",
		ap.ParcelID, ap.AssetID, ap.Position, ap.Rotation, ap.Scale, ap.PlacedBy,
	).Scan(&ap.ID, &ap.CreatedAt, &ap.UpdatedAt)
	return err
}

func (r *AssetPlacementRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, "DELETE FROM asset_placements WHERE id = $1", id)
	return err
}

type PackageRepository struct {
	db *pgxpool.Pool
}

func NewPackageRepository(pool *pgxpool.Pool) *PackageRepository {
	return &PackageRepository{db: pool}
}

func (r *PackageRepository) GetAll(ctx context.Context) ([]*Package, error) {
	rows, err := r.db.Query(ctx,
		"SELECT id, name, type, version, size, description, content_path, dependencies, created_at FROM packages ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var packages []*Package
	for rows.Next() {
		var p Package
		if err := rows.Scan(&p.ID, &p.Name, &p.Type, &p.Version, &p.Size, &p.Description, &p.ContentPath, &p.Dependencies, &p.CreatedAt); err != nil {
			return nil, err
		}
		packages = append(packages, &p)
	}
	return packages, rows.Err()
}

func (r *PackageRepository) GetByID(ctx context.Context, id string) (*Package, error) {
	var p Package
	err := r.db.QueryRow(ctx,
		"SELECT id, name, type, version, size, description, content_path, dependencies, created_at FROM packages WHERE id = $1",
		id,
	).Scan(&p.ID, &p.Name, &p.Type, &p.Version, &p.Size, &p.Description, &p.ContentPath, &p.Dependencies, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PackageRepository) GetByType(ctx context.Context, pkgType string) ([]*Package, error) {
	rows, err := r.db.Query(ctx,
		"SELECT id, name, type, version, size, description, content_path, dependencies, created_at FROM packages WHERE type = $1 ORDER BY created_at DESC",
		pkgType,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var packages []*Package
	for rows.Next() {
		var p Package
		if err := rows.Scan(&p.ID, &p.Name, &p.Type, &p.Version, &p.Size, &p.Description, &p.ContentPath, &p.Dependencies, &p.CreatedAt); err != nil {
			return nil, err
		}
		packages = append(packages, &p)
	}
	return packages, rows.Err()
}

func (r *PackageRepository) Create(ctx context.Context, p *Package) error {
	err := r.db.QueryRow(ctx,
		"INSERT INTO packages (name, type, version, size, description, content_path, dependencies) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, created_at",
		p.Name, p.Type, p.Version, p.Size, p.Description, p.ContentPath, p.Dependencies,
	).Scan(&p.ID, &p.CreatedAt)
	return err
}
