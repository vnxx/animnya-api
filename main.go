package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
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
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET",
	}))
	app.Use(cache.New(cache.Config{
		ExpirationGenerator: func(c *fiber.Ctx, cfg *cache.Config) time.Duration {
			expiration := time.Minute * 10
			newCacheTime, _ := strconv.Atoi(c.GetRespHeader("Cache-Time", fmt.Sprintf("%.0f", expiration.Seconds())))
			return time.Second * time.Duration(newCacheTime)
		},
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.Path()
		},
	}))

	db := db.New()
	fetch := lib.NewFetcher()
	handler := handler.New(fetch, db)

	router.SetupRoutes(app, handler)
	app.Listen(fmt.Sprintf(":%s", os.Getenv("PORT")))
}
