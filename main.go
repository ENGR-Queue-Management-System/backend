package main

import (
	"log"
	"net/http"
	"os"
	"src/api"
	"src/db"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
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

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete},
	}))
	e.Use(middleware.BodyLimit("2M"))

	apiV1 := e.Group("/api/v1")
	apiV1.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "API is working!")
	})
	api.RegisterRoutes(apiV1, dbConn)

	e.Logger.Fatal(e.Start(":" + port))
}
