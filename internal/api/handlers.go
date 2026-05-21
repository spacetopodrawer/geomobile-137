package api

import (
	"cadastre_ia/internal/config"
	"cadastre_ia/internal/service"
	"cadastre_ia/internal/storage"
	"cadastre_ia/internal/websocket"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// ==================== AUTH ENDPOINTS ====================

func registerUser(store *storage.Storage, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email    string `json:"email" binding:"required,email"`
			Password string `json:"password" binding:"required,min=8"`
			Name     string `json:"name" binding:"required"`
		}

		if err := c.BindJSON(&req); err != nil {
			log.Printf("❌ [AUTH] Registration binding failed: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		authService := service.NewAuthService(storage.NewUserRepository(store.GetDB()), cfg)
		user, token, err := authService.Register(c.Request.Context(), req.Email, req.Password, req.Name)
		if err != nil {
			log.Printf("❌ [AUTH] Registration failed: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"user": gin.H{
				"id":    user.ID,
				"email": user.Email,
				"name":  user.Name,
			},
			"token": token,
		})
	}
}

func loginUser(store *storage.Storage, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email    string `json:"email" binding:"required,email"`
			Password string `json:"password" binding:"required"`
		}

		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		authService := service.NewAuthService(storage.NewUserRepository(store.GetDB()), cfg)
		user, token, err := authService.Login(c.Request.Context(), req.Email, req.Password)
		if err != nil {
			log.Printf("⚠️  [AUTH] Login failed: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"user": gin.H{
				"id":    user.ID,
				"email": user.Email,
				"name":  user.Name,
			},
			"token": token,
		})
	}
}

func refreshToken(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
			return
		}

		// In production, verify user still exists and hasn't been suspended
		authService := service.NewAuthService(nil, cfg)
		newToken, err := authService.GenerateToken(userID.(string), "", 24*60*60)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"token": newToken})
	}
}

// ==================== PARCEL ENDPOINTS ====================

func listParcels(store *storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		workspaceID := c.Query("workspace_id")

		if workspaceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "workspace_id query parameter required"})
			return
		}

		log.Printf("📋 [API] Listing parcels for workspace: %s user: %s", workspaceID, userID)

		parcelRepo := storage.NewParcelRepository(store.GetDB())
		parcels, err := parcelRepo.GetByWorkspace(c.Request.Context(), workspaceID)
		if err != nil {
			log.Printf("❌ [API] Failed to list parcels: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list parcels"})
			return
		}

		if parcels == nil {
			parcels = []*storage.Parcel{}
		}

		c.JSON(http.StatusOK, gin.H{
			"parcels": parcels,
			"total":   len(parcels),
		})
	}
}

func createParcel(store *storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			WorkspaceID string `json:"workspace_id" binding:"required"`
			ParcelNum   string `json:"parcel_number" binding:"required"`
			Geometry    string `json:"geometry" binding:"required"`
		}

		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		userID, _ := c.Get("user_id")
		log.Printf("➕ [API] Creating parcel: workspace=%s user=%s", req.WorkspaceID, userID)

		parcelRepo := storage.NewParcelRepository(store.GetDB())
		// Create parcel
		parcel := &storage.Parcel{
			WorkspaceID: storage.MustParseUUID(req.WorkspaceID),
			ParcelNum:   req.ParcelNum,
			Geometry:    req.Geometry,
			Version:     1,
		}

		if err := parcelRepo.Create(c.Request.Context(), parcel); err != nil {
			log.Printf("❌ [API] Failed to create parcel: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create parcel"})
			return
		}

		c.JSON(http.StatusCreated, parcel)
	}
}

func getParcel(store *storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		log.Printf("🔍 [API] Getting parcel: %s", id)

		parcelRepo := storage.NewParcelRepository(store.GetDB())
		parcel, err := parcelRepo.GetByID(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "parcel not found"})
			return
		}

		c.JSON(http.StatusOK, parcel)
	}
}

func updateParcel(store *storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		log.Printf("✏️  [API] Updating parcel: %s", id)

		var req struct {
			Geometry string `json:"geometry"`
		}

		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		parcelRepo := storage.NewParcelRepository(store.GetDB())
		parcel, err := parcelRepo.GetByID(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "parcel not found"})
			return
		}

		parcel.Geometry = req.Geometry
		parcel.Version++
		if err := parcelRepo.Update(c.Request.Context(), parcel); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update parcel"})
			return
		}

		c.JSON(http.StatusOK, parcel)
	}
}

func deleteParcel(store *storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		log.Printf("🗑️  [API] Deleting parcel: %s", id)

		parcelRepo := storage.NewParcelRepository(store.GetDB())
		if err := parcelRepo.Delete(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete parcel"})
			return
		}

		c.Status(http.StatusNoContent)
	}
}

// ==================== CAD ENDPOINTS ====================

func convertCAD(store *storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		parcelID := c.PostForm("parcel_id")

		if parcelID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "parcel_id form field required"})
			return
		}

		file, err := c.FormFile("file")
		if err != nil {
			log.Printf("❌ [API] CAD file upload failed: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "file form field required"})
			return
		}

		log.Printf("📤 [API] CAD conversion initiated: parcel=%s user=%s filename=%s size=%d",
			parcelID, userID, file.Filename, file.Size)

		// Read file content
		fileContent, err := file.Open()
		if err != nil {
			log.Printf("❌ [API] Failed to open file: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read file"})
			return
		}
		defer fileContent.Close()

		fileData := make([]byte, file.Size)
		_, err = fileContent.Read(fileData)
		if err != nil {
			log.Printf("❌ [API] Failed to read file content: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read file"})
			return
		}

		// Detect format from filename extension
		format := "unknown"
		if strings.Contains(file.Filename, ".dxf") {
			format = "dxf"
		} else if strings.Contains(file.Filename, ".dwg") {
			format = "dwg"
		} else if strings.Contains(file.Filename, ".json") {
			format = "geojson"
		}

		fileRepo := storage.NewCADFileRepository(store.GetDB())
		cadService := service.NewCADConverterService(fileRepo)

		// Convert and store
		cadFile, err := cadService.ConvertFile(c.Request.Context(), fileData, format, parcelID, userID.(string), file.Filename)
		if err != nil {
			log.Printf("❌ [API] CAD conversion failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusAccepted, gin.H{
			"file_id":              cadFile.ID,
			"conversion_status":    cadFile.ConversionStatus,
			"original_filename":    cadFile.OriginalFilename,
			"message":              "conversion in progress, check status via GET /cad/{file_id}/status",
		})
	}
}

func checkConversionStatus(store *storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		fileID := c.Param("id")
		log.Printf("🔍 [API] Checking CAD conversion status: %s", fileID)

		fileRepo := storage.NewCADFileRepository(store.GetDB())
		cadFile, err := fileRepo.GetByID(c.Request.Context(), fileID)
		if err != nil {
			log.Printf("❌ [API] CAD file not found: %v", err)
			c.JSON(http.StatusNotFound, gin.H{"error": "CAD file not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"file_id":              cadFile.ID,
			"original_filename":    cadFile.OriginalFilename,
			"conversion_status":    cadFile.ConversionStatus,
			"converted_geometry":   cadFile.ConvertedGeometry,
			"conversion_error":     cadFile.ConversionError,
			"created_at":           cadFile.CreatedAt,
			"updated_at":           cadFile.UpdatedAt,
		})
	}
}

func exportCAD(store *storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		parcelID := c.Param("parcel_id")
		format := c.Query("format")

		if format == "" {
			format = "geojson"
		}

		log.Printf("📥 [API] CAD export: parcel=%s format=%s", parcelID, format)

		fileRepo := storage.NewCADFileRepository(store.GetDB())
		cadService := service.NewCADConverterService(fileRepo)

		data, err := cadService.ExportToFormat(c.Request.Context(), parcelID, format)
		if err != nil {
			log.Printf("❌ [API] CAD export failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Generate filename based on parcelID and format
		filename := fmt.Sprintf("parcel_%s.%s", parcelID, format)
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		c.Header("Content-Type", "application/octet-stream")
		c.Data(http.StatusOK, "application/octet-stream", data)
	}
}

// ==================== ASSET ENDPOINTS ====================

func listAssets(store *storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("📦 [API] Listing all 3D assets")

		assetRepo := storage.NewAsset3DRepository(store.GetDB())
		assetService := service.NewAssetService(assetRepo, nil)

		assets, err := assetService.ListAssets(c.Request.Context())
		if err != nil {
			log.Printf("❌ [API] Failed to list assets: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list assets"})
			return
		}

		if assets == nil {
			assets = []*storage.Asset3D{}
		}

		c.JSON(http.StatusOK, gin.H{
			"assets": assets,
			"total":  len(assets),
		})
	}
}

func placeAsset(store *storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var req struct {
			ParcelID string  `json:"parcel_id" binding:"required"`
			AssetID  string  `json:"asset_id" binding:"required"`
			Position string  `json:"position" binding:"required"`
			Rotation float64 `json:"rotation" binding:"required"`
			Scale    float64 `json:"scale" binding:"required"`
		}

		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		log.Printf("📍 [API] Placing asset: parcel=%s asset=%s user=%s", req.ParcelID, req.AssetID, userID)

		assetRepo := storage.NewAsset3DRepository(store.GetDB())
		placementRepo := storage.NewAssetPlacementRepository(store.GetDB())
		assetService := service.NewAssetService(assetRepo, placementRepo)

		placement, err := assetService.PlaceAsset(c.Request.Context(), req.ParcelID, req.AssetID,
			userID.(string), req.Position, req.Rotation, req.Scale)
		if err != nil {
			log.Printf("❌ [API] Failed to place asset: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, placement)
	}
}

func downloadAsset(store *storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		assetID := c.Param("id")
		log.Printf("⬇️  [API] Downloading asset: %s", assetID)

		assetRepo := storage.NewAsset3DRepository(store.GetDB())
		assets, err := assetRepo.GetAll(c.Request.Context())
		if err != nil {
			log.Printf("❌ [API] Failed to fetch asset: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch asset"})
			return
		}

		var asset *storage.Asset3D
		for _, a := range assets {
			if a.ID.String() == assetID {
				asset = a
				break
			}
		}

		if asset == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "asset not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"asset_id":     asset.ID,
			"name":         asset.Name,
			"model_url":    asset.ModelURL,
			"model_format": asset.ModelFormat,
			"category":     asset.Category,
			"scale":        asset.Scale,
			"preview_icon": asset.PreviewIcon,
		})
	}
}

func removeAssetPlacement(store *storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		placementID := c.Param("id")
		log.Printf("🗑️  [API] Removing asset placement: %s", placementID)

		placementRepo := storage.NewAssetPlacementRepository(store.GetDB())
		assetService := service.NewAssetService(nil, placementRepo)

		if err := assetService.RemovePlacement(c.Request.Context(), placementID); err != nil {
			log.Printf("❌ [API] Failed to remove placement: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.Status(http.StatusNoContent)
	}
}

// ==================== CHAT ENDPOINTS ====================

func listMessages(store *storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		workspaceID := c.Query("workspace_id")
		limitStr := c.DefaultQuery("limit", "50")
		offsetStr := c.DefaultQuery("offset", "0")

		if workspaceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "workspace_id query parameter required"})
			return
		}

		log.Printf("📨 [API] Listing messages for workspace: %s user: %s", workspaceID, userID)

		msgRepo := storage.NewMessageRepository(store.GetDB())
		chatService := service.NewChatService(msgRepo, nil)

		messages, err := chatService.GetMessages(c.Request.Context(), workspaceID, limitStr, offsetStr)
		if err != nil {
			log.Printf("❌ [API] Failed to list messages: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if messages == nil {
			messages = []*storage.Message{}
		}

		c.JSON(http.StatusOK, gin.H{
			"messages": messages,
			"total":    len(messages),
		})
	}
}

func sendMessage(store *storage.Storage, hub *websocket.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var req struct {
			WorkspaceID string `json:"workspace_id" binding:"required"`
			Content     string `json:"content" binding:"required"`
			ThreadID    string `json:"thread_id"`
		}

		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		log.Printf("💬 [API] Sending message: workspace=%s user=%s thread=%s",
			req.WorkspaceID, userID, req.ThreadID)

		msgRepo := storage.NewMessageRepository(store.GetDB())
		chatService := service.NewChatService(msgRepo, hub)

		message, err := chatService.SendMessage(c.Request.Context(), req.WorkspaceID, userID.(string), req.Content, req.ThreadID)
		if err != nil {
			log.Printf("❌ [API] Failed to send message: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, message)
	}
}

func getThreadMessages(store *storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		threadID := c.Param("thread_id")
		log.Printf("🧵 [API] Fetching thread messages: thread=%s", threadID)

		msgRepo := storage.NewMessageRepository(store.GetDB())
		messages, err := msgRepo.GetByThread(c.Request.Context(), threadID)
		if err != nil {
			log.Printf("❌ [API] Failed to fetch thread messages: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch thread messages"})
			return
		}

		if messages == nil {
			messages = []*storage.Message{}
		}

		c.JSON(http.StatusOK, gin.H{
			"thread_id": threadID,
			"messages":  messages,
			"total":     len(messages),
		})
	}
}

// ==================== FILE ENDPOINTS ====================

func uploadFile(store *storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		parcelID := c.PostForm("parcel_id")

		if parcelID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "parcel_id form field required"})
			return
		}

		file, err := c.FormFile("file")
		if err != nil {
			log.Printf("❌ [API] File upload failed: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "file form field required"})
			return
		}

		log.Printf("📤 [API] File upload: parcel=%s user=%s filename=%s size=%d",
			parcelID, userID, file.Filename, file.Size)

		// Read file content
		fileContent, err := file.Open()
		if err != nil {
			log.Printf("❌ [API] Failed to open file: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read file"})
			return
		}
		defer fileContent.Close()

		fileData := make([]byte, file.Size)
		_, err = fileContent.Read(fileData)
		if err != nil {
			log.Printf("❌ [API] Failed to read file content: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read file"})
			return
		}

		fvRepo := storage.NewFileVersionRepository(store.GetDB())
		fileService := service.NewFileService(fvRepo)

		fileVersion, err := fileService.UploadFile(c.Request.Context(), parcelID, userID.(string), file.Filename, fileData)
		if err != nil {
			log.Printf("❌ [API] Upload failed: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, fileVersion)
	}
}

func downloadFile(store *storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		fileVersionID := c.Param("id")
		log.Printf("Download file: %s", fileVersionID)

		// In production: fetch file from S3/storage by fileVersionID
		// For now, return 404 as storage implementation is pending
		c.JSON(http.StatusNotFound, gin.H{"error": "file storage integration pending"})
	}
}

func getFileVersions(store *storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		parcelID := c.Param("parcel_id")
		log.Printf("📄 [API] Fetching file versions: parcel=%s", parcelID)

		fvRepo := storage.NewFileVersionRepository(store.GetDB())
		versions, err := fvRepo.GetByParcel(c.Request.Context(), parcelID)
		if err != nil {
			log.Printf("❌ [API] Failed to fetch file versions: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch file versions"})
			return
		}

		if versions == nil {
			versions = []*storage.FileVersion{}
		}

		c.JSON(http.StatusOK, gin.H{
			"parcel_id": parcelID,
			"versions":  versions,
			"total":     len(versions),
		})
	}
}

// ==================== ANIMATION ENDPOINTS ====================

func listAnimations(store *storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		parcelID := c.Query("parcel_id")

		if parcelID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "parcel_id query parameter required"})
			return
		}

		log.Printf("Listing animations for parcel: %s", parcelID)

		// In production: add GetByParcel method to AnimationRepository
		// For now, return empty list
		c.JSON(http.StatusOK, gin.H{
			"parcel_id":  parcelID,
			"animations": []*storage.AnimationSequence{},
			"total":      0,
		})
	}
}

func createAnimation(store *storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			ParcelID string          `json:"parcel_id" binding:"required"`
			Name     string          `json:"name" binding:"required"`
			Type     string          `json:"type" binding:"required"`
			Duration float64         `json:"duration" binding:"required"`
			Keyframes []byte          `json:"keyframes"`
		}

		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		log.Printf("🎬 [API] Creating animation: parcel=%s name=%s type=%s", req.ParcelID, req.Name, req.Type)

		animRepo := storage.NewAnimationRepository(store.GetDB())
		animService := service.NewAnimationService(animRepo)

		animation, err := animService.CreateAnimation(c.Request.Context(), req.Name, req.Type, req.Duration, req.Keyframes)
		if err != nil {
			log.Printf("❌ [API] Failed to create animation: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, animation)
	}
}

func updateAnimation(store *storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		animID := c.Param("id")

		var req struct {
			Name     string   `json:"name"`
			Duration float64  `json:"duration"`
			Keyframes []byte   `json:"keyframes"`
		}

		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		log.Printf("✏️  [API] Updating animation: %s", animID)

		animRepo := storage.NewAnimationRepository(store.GetDB())
		animService := service.NewAnimationService(animRepo)

		animation, err := animService.UpdateAnimation(c.Request.Context(), animID, req.Name, req.Duration, req.Keyframes)
		if err != nil {
			log.Printf("❌ [API] Failed to update animation: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, animation)
	}
}

// ==================== PACKAGE ENDPOINTS ====================

func listPackages(store *storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("📦 [API] Listing all packages")

		pkgRepo := storage.NewPackageRepository(store.GetDB())
		pkgService := service.NewPackageService(pkgRepo)

		packages, err := pkgService.ListPackages(c.Request.Context())
		if err != nil {
			log.Printf("❌ [API] Failed to list packages: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list packages"})
			return
		}

		if packages == nil {
			packages = []*storage.Package{}
		}

		c.JSON(http.StatusOK, gin.H{
			"packages": packages,
			"total":    len(packages),
		})
	}
}

func createPackage(store *storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name         string   `json:"name" binding:"required"`
			Type         string   `json:"type" binding:"required"`
			Version      string   `json:"version" binding:"required"`
			Size         int64    `json:"size" binding:"required"`
			Description  string   `json:"description"`
			ContentPath  string   `json:"content_path" binding:"required"`
			Dependencies []string `json:"dependencies"`
		}

		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		log.Printf("📦 [API] Creating package: name=%s type=%s version=%s", req.Name, req.Type, req.Version)

		pkgRepo := storage.NewPackageRepository(store.GetDB())
		pkgService := service.NewPackageService(pkgRepo)

		pkg, err := pkgService.CreatePackage(c.Request.Context(), req.Name, req.Type, req.Version,
			req.Size, req.Description, req.ContentPath, req.Dependencies)
		if err != nil {
			log.Printf("❌ [API] Failed to create package: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, pkg)
	}
}

func downloadPackage(store *storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		packageID := c.Param("id")
		log.Printf("⬇️  [API] Downloading package: %s", packageID)

		pkgRepo := storage.NewPackageRepository(store.GetDB())
		pkgService := service.NewPackageService(pkgRepo)

		pkg, err := pkgService.DownloadPackage(c.Request.Context(), packageID)
		if err != nil {
			log.Printf("❌ [API] Failed to download package: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"package_id":    pkg.ID,
			"name":          pkg.Name,
			"type":          pkg.Type,
			"version":       pkg.Version,
			"size":          pkg.Size,
			"content_path":  pkg.ContentPath,
			"download_url":  fmt.Sprintf("/api/packages/%s/content", pkg.ID),
			"message":       "download via signed URL or content endpoint",
		})
	}
}

// ==================== WEBSOCKET ====================

func handleWebSocketUpgrade(hub *websocket.Hub, secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
			return
		}

		workspaceID := c.Query("workspace_id")
		if workspaceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "workspace_id query parameter required"})
			return
		}

		log.Printf("🔌 [WS] Upgrade request: user=%s workspace=%s", userID, workspaceID)

		// Upgrade HTTP connection to WebSocket
		ws, err := websocket.Upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("❌ [WS] Upgrade failed: %v", err)
			return
		}

		// Create client and register with hub using proper API
		deviceID := workspaceID // Using workspace_id as deviceID for now
		client := hub.RegisterConnection(ws, deviceID, userID.(string))

		log.Printf("✅ [WS] Client connected: user=%s device=%s", userID, deviceID)

		// Start reading and writing goroutines
		go client.ReadMessages()
		go client.WriteMessages()
	}
}
