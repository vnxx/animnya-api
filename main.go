package main

import (
	"fmt"
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
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	app := fiber.New()
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET",
	}))
	app.Use(cache.New(cache.Config{
		ExpirationGenerator: func(c *fiber.Ctx, cfg *cache.Config) time.Duration {
			expiration := time.Hour * 2
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
