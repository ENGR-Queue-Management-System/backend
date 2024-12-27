package main

import (
	"log"
	"net/http"
	"os"
	"src/api"
	"src/db"
	"src/helpers"
	"time"

	"github.com/gin-gonic/gin"
	socketio "github.com/googollee/go-socket.io"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("PORT environment variable is not set")
	}

	dbConn := db.ConnectDB()
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	defer dbConn.Close()

	helpers.StartCounterStatusUpdater(dbConn, time.Minute)

	router := gin.Default()
	router.Use(func(c *gin.Context) {
		if c.Request.ContentLength > 2*1024*1024 { // 2 MB
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "Request body too large",
			})
			return
		}
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept, Authorization, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusOK)
			return
		}
		c.Request.Header.Del("Origin")
		c.Next()
	})
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	server := socketio.NewServer(nil)
	server.OnConnect(helpers.SOCKET, func(s socketio.Conn) error {
		s.SetContext("")
		println("New connection:", s.ID())
		return nil
	})
	server.OnDisconnect(helpers.SOCKET, func(s socketio.Conn, reason string) {
		println("Disconnected:", s.ID(), reason)
	})
	// server.OnEvent(helpers.SOCKET, "setLoginNotCmu", func(s socketio.Conn, msg string) {})
	// server.OnEvent(helpers.SOCKET, "addQueue", func(s socketio.Conn, msg string) {})

	go func() {
		if err := server.Serve(); err != nil {
			log.Fatal(err)
		}
	}()
	defer server.Close()

	router.GET("/socket.io/*any", gin.WrapH(server))
	router.POST("/socket.io/*any", gin.WrapH(server))

	apiV1 := router.Group("/api/v1")
	api.RegisterRoutes(apiV1, dbConn, server)

	log.Fatal(router.Run(":" + port))
}
