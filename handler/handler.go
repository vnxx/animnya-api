package handler

import (
	"animenya.site/cache"
	"animenya.site/db"
	"animenya.site/lib"
)

type HandlerInterface interface {
	Cache() cache.CacheInterface
	Fetch() lib.FetcherInterface
}

type Handler struct {
	Cache   cache.CacheInterface
	Fetcher lib.FetcherInterface
	DB      db.DBInterface
}

func New(cache cache.CacheInterface, fetch lib.FetcherInterface, db db.DBInterface) *Handler {
	return &Handler{
		Cache:   cache,
		Fetcher: fetch,
		DB:      db,
	}
}
