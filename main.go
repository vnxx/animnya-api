package main

import (
	"fmt"
	"log"
	"os"

	"animenya.site/cache"
	"animenya.site/db"
	"animenya.site/handler"
	"animenya.site/lib"
	"animenya.site/router"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		log.Fatalf("Error loading .env file: \n%s", err)
	}

	app := fiber.New()
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept",
	}))

	db := db.New()
	cache := cache.New()
	fetch := lib.NewFetcher()
	handler := handler.New(cache, fetch, db)

	router.SetupRoutes(app, handler)
	app.Listen(fmt.Sprintf(":%s", os.Getenv("PORT")))
}
