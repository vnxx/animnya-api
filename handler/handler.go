package handler

import (
	"animenya.site/db"
	"animenya.site/lib"
)

type HandlerInterface interface {
	Fetch() lib.FetcherInterface
}

type Handler struct {
	Fetcher lib.FetcherInterface
	DB      db.DBInterface
}

func New(fetch lib.FetcherInterface, db db.DBInterface) *Handler {
	return &Handler{
		Fetcher: fetch,
		DB:      db,
	}
}
