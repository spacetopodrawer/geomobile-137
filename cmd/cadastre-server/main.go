package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	gorillaWs "github.com/gorilla/websocket"

	internalAPI "cadastre_ia/internal/api"
	wsHub "cadastre_ia/internal/websocket"
	"cadastre_ia/pkg/storage"
)

var (
	port   = flag.String("port", "8080", "Server port")
	dbMode = flag.String("db", "mock", "Database mode: mock | postgres")
	dbConn = flag.String("db-conn", "", "PostgreSQL DSN (required when -db=postgres)")
)

var upgrader = gorillaWs.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func main() {
	flag.Parse()
	fmt.Println("+============================================================+")
	fmt.Println("|  GEO-MOBILE137  CADASTRAL SERVER  v3.0.0                   |")
	fmt.Println("|  Phase 3 - Real-Time Collaboration & WebHooks              |")
	fmt.Println("+============================================================+")
	fmt.Printf("   Port    : %s\n", *port)
	fmt.Printf("   DB mode : %s\n", *dbMode)
	fmt.Println("\n[*] Initializing components...")

	var dbConnected bool
	if *dbMode == "postgres" {
		if *dbConn == "" {
			log.Fatal("ERROR: -db-conn required for -db=postgres")
		}
		fmt.Printf("   DB conn : %s\n", *dbConn)
		if _, err := storage.Connect(*dbConn); err != nil {
			log.Printf("WARNING: PostgreSQL unavailable (%v)", err)
		} else {
			dbConnected = true
			fmt.Println("     OK PostgreSQL connected")
		}
	} else {
		fmt.Println("     OK Mock database ready")
		dbConnected = true
	}

	fmt.Println("  --> Initializing WebSocket Hub...")
	hub := wsHub.NewHub()
	go hub.Run()
	fmt.Println("     OK WebSocket Hub running")

	p3 := internalAPI.NewPhase3Handlers(hub)
	fmt.Println("     OK Phase 3 handlers ready")

	if dbConnected {
		fmt.Println("   Database : CONNECTED")
	}
	fmt.Printf("   WS Hub  : RUNNING (%d clients)\n", hub.GetConnectedClients())

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization,X-User-ID")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":     "ok",
			"version":    "3.0.0",
			"phase":      "3",
			"db_mode":    *dbMode,
			"db":         dbStatus(dbConnected),
			"ws_clients": hub.GetConnectedClients(),
			"timestamp":  time.Now().Format(time.RFC3339),
		})
	})

	r.GET("/ws/geometry", func(c *gin.Context) {
		dID := c.Query("device_id")
		uID := c.Query("user_id")
		if dID == "" {
			dID = "device-" + uuid.New().String()[:8]
		}
		if uID == "" {
			uID = "user-" + uuid.New().String()[:8]
		}
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("[WS] upgrade failed: %v", err)
			return
		}
		client := hub.RegisterConnection(conn, dID, uID)
		log.Printf("[WS] CONNECTED device=%s user=%s", dID, uID)
		go client.ReadMessages()
		go client.WriteMessages()
	})

	api := r.Group("/api/v1")
	api.GET("/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"version":     "3.0.0",
			"phase":       "3",
			"db":          dbStatus(dbConnected),
			"ws_clients":  hub.GetConnectedClients(),
			"server_time": time.Now().UTC(),
		})
	})

	cad := api.Group("/cadastre")
	cad.GET("/tiles/:z/:x/:y", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"tile":   fmt.Sprintf("%s/%s/%s", c.Param("z"), c.Param("x"), c.Param("y")),
			"format": "png",
		})
	})
	cad.POST("/convert", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"conversion_id": uuid.New().String(), "status": "queued"})
	})
	cad.POST("/decode", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"decode_id": uuid.New().String(), "result": nil})
	})
	cad.POST("/validate", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"valid": true, "errors": []interface{}{}})
	})
	cad.GET("/legend", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"legend": []interface{}{}})
	})
	cad.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "subsystem": "cadastre"})
	})

	api.GET("/parcels", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"parcels": []interface{}{}, "total": 0})
	})
	api.POST("/parcels", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"parcel_id": uuid.New().String(), "created_at": time.Now().UTC()})
	})
	api.GET("/parcels/:id", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"parcel_id": c.Param("id")})
	})
	api.PUT("/parcels/:id", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"parcel_id": c.Param("id"), "updated_at": time.Now().UTC()})
	})
	api.DELETE("/parcels/:id", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"parcel_id": c.Param("id"), "deleted": true})
	})

	sync := api.Group("/sync")
	sync.POST("/init", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"sync_id": uuid.New().String(), "initialized": true})
	})
	sync.GET("/state", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"device_id": c.Query("device_id"), "sync_version": 0})
	})
	sync.POST("/upload", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"upload_id": uuid.New().String(), "accepted": true})
	})
	sync.GET("/download", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"device_id": c.Query("device_id"), "changes": []interface{}{}})
	})
	sync.POST("/resolve", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"resolved": true, "strategy": "last_write_wins"})
	})
	sync.GET("/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"device_id": c.Query("device_id"), "status": "synced"})
	})

	rtk := api.Group("/rtk")
	rtk.POST("/enable", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"state": "INITIALIZATION", "enabled_at": time.Now().UTC()})
	})
	rtk.POST("/disable", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"state": "DISABLED"})
	})
	rtk.GET("/state", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"device_id": c.Query("device_id"), "state": "DISABLED"})
	})
	rtk.POST("/submit-position", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"applied": true, "timestamp": time.Now().UTC()})
	})

	garmin := api.Group("/garmin")
	garmin.POST("/pair", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "paired", "paired_at": time.Now().UTC()})
	})
	garmin.GET("/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "not_paired"})
	})
	garmin.POST("/sensors", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"recorded": true})
	})
	garmin.POST("/disconnect", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"disconnected": true})
	})

	// Phase 3A
	api.GET("/ws/status", p3.HubStatus)
	api.GET("/presence", p3.GetPresence)
	api.POST("/activity", p3.LogActivity)
	api.GET("/activity", p3.GetActivityLog)
	api.POST("/parcels/:id/lock", p3.AcquireLock)
	api.DELETE("/parcels/:id/lock", p3.ReleaseLock)

	// Phase 3B
	api.GET("/notifications", p3.ListNotifications)
	api.PATCH("/notifications/:id/read", p3.MarkNotificationRead)
	api.POST("/notifications/broadcast", p3.BroadcastNotification)
	api.GET("/webhooks", p3.ListWebhooks)
	api.POST("/webhooks", p3.CreateWebhook)
	api.DELETE("/webhooks/:id", p3.DeleteWebhook)

	fmt.Printf("\n[HTTP] http://0.0.0.0:%s\n", *port)
	fmt.Printf("[WS]   ws://localhost:%s/ws/geometry\n\n", *port)
	if err := r.Run(":" + *port); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func dbStatus(connected bool) string {
	if connected {
		return "connected"
	}
	return "unavailable"
}
