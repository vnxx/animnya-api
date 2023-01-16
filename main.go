package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"animenya.site/db"
	"animenya.site/handler"
	"animenya.site/lib"
	"animenya.site/router"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cache"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		log.Fatalf("Error loading .env file: \n%s", err)
	}

	app := fiber.New()
	app.Use(cache.New(cache.Config{
		Expiration: time.Minute * 15,
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "*",
	}))

	db := db.New()
	fetch := lib.NewFetcher()
	handler := handler.New(fetch, db)

	router.SetupRoutes(app, handler)
	app.Listen(fmt.Sprintf(":%s", os.Getenv("PORT")))
}
