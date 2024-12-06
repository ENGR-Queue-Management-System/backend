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

	dbConn := db.ConnectDB()
	defer dbConn.Close()

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete},
	}))

	apiV1 := e.Group("/api/v1")
	api.RegisterRoutes(apiV1, dbConn)

	e.Logger.Fatal(e.Start(":" + os.Getenv("PORT")))
}
