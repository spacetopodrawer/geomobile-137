package api

import (
	"cadastre_ia/internal/config"
	"cadastre_ia/internal/storage"
	"cadastre_ia/internal/websocket"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(router *gin.Engine, store *storage.Storage, wsHub *websocket.Hub, cfg *config.Config) {
	// Apply middleware
	router.Use(corsMiddleware())
	router.Use(loggingMiddleware())
	router.Use(errorHandlingMiddleware())

	// Public routes (no JWT required)
	public := router.Group("/api/v1")
	{
		// Auth routes (public)
		authGroup := public.Group("/auth")
		{
			authGroup.POST("/register", registerUser(store, cfg))
			authGroup.POST("/login", loginUser(store, cfg))
			authGroup.POST("/refresh", refreshToken(cfg))
		}

		// Document scanning routes (public for initial identification)
		documentHandlers := NewDocumentHandlers(store, cfg)
		documentGroup := public.Group("/documents")
		{
			documentGroup.POST("/scan", documentHandlers.ScanDocument(store, cfg))
		}

		// Facial recognition login (public)
		identityGroup := public.Group("/identity")
		{
			identityGroup.POST("/login-facial", documentHandlers.FacialLogin(store, cfg))
		}
	}

	// Protected routes (JWT required)
	v1 := router.Group("/api/v1")
	{
		v1.Use(jwtMiddleware(cfg.JWTSecret))

		// Document routes (protected)
		documentHandlers := NewDocumentHandlers(store, cfg)
		documentGroupProtected := v1.Group("/documents")
		{
			documentGroupProtected.GET("/my-scans", documentHandlers.GetDocumentHistory(store))
			documentGroupProtected.GET("/properties", documentHandlers.GetDocumentSummary(store))
			documentGroupProtected.POST("/:scan_id/verify", documentHandlers.ConfirmDocumentIdentification(store))
		}

		// Identity routes (protected)
		identityGroupProtected := v1.Group("/identity")
		{
			identityGroupProtected.POST("/register-face", documentHandlers.RegisterFacePhoto(store, cfg))
		}

		// Parcel routes
		parcelGroup := v1.Group("/parcels")
		{
			parcelGroup.GET("", listParcels(store))
			parcelGroup.POST("", createParcel(store))
			parcelGroup.GET("/:id", getParcel(store))
			parcelGroup.PATCH("/:id", updateParcel(store))
			parcelGroup.DELETE("/:id", deleteParcel(store))
		}

		// CAD routes
		cadGroup := v1.Group("/cad")
		{
			cadGroup.POST("/convert", convertCAD(store))
			cadGroup.GET("/:id/status", checkConversionStatus(store))
			cadGroup.GET("/:id/export", exportCAD(store))  // Changed from :parcel_id to :id to avoid conflict
		}

		// Asset routes
		assetGroup := v1.Group("/assets")
		{
			assetGroup.GET("", listAssets(store))
			assetGroup.POST("/place", placeAsset(store))
			assetGroup.GET("/:id", downloadAsset(store))
			assetGroup.DELETE("/placements/:id", removeAssetPlacement(store))
		}

		// Message routes
		messageGroup := v1.Group("/messages")
		{
			messageGroup.GET("", listMessages(store))
			messageGroup.POST("", sendMessage(store, wsHub))
			messageGroup.GET("/threads/:thread_id", getThreadMessages(store))
		}

		// File routes
		fileGroup := v1.Group("/files")
		{
			fileGroup.POST("/upload", uploadFile(store))
			fileGroup.GET("/:id", downloadFile(store))
			fileGroup.GET("/parcel/:parcel_id/versions", getFileVersions(store))
		}

		// Animation routes
		animationGroup := v1.Group("/animations")
		{
			animationGroup.GET("", listAnimations(store))
			animationGroup.POST("", createAnimation(store))
			animationGroup.PATCH("/:id", updateAnimation(store))
		}

		// Package routes
		packageGroup := v1.Group("/packages")
		{
			packageGroup.GET("", listPackages(store))
			packageGroup.POST("", createPackage(store))
			packageGroup.GET("/:id", downloadPackage(store))
		}
	}

	// WebSocket endpoint (JWT auth via query token)
	router.GET("/ws", handleWebSocketUpgrade(wsHub, cfg.JWTSecret))
}
