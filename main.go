package main

import (
	"log"
	"net/http"
	"os"
	"src/api"
	"src/db"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
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

	dbConn := db.ConnectDB().Debug()
	defer func() {
		sqlDB, _ := dbConn.DB()
		sqlDB.Close()
	}()

	db.StartCounterStatusUpdater(dbConn, time.Minute)

	hub := api.NewHub()
	go hub.Run()

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
		c.Next()
	})
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	router.GET("/ws", func(c *gin.Context) {
		api.ServeWs(hub, c.Writer, c.Request)
	})

	apiV1 := router.Group("/api/v1")
	api.RegisterRoutes(apiV1, dbConn, hub)

	log.Fatal(router.Run(":" + port))
}
