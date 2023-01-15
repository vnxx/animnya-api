package model

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"animenya.site/data"
	"animenya.site/db"
)

type Anime struct {
	ID            int        `json:"id"`
	Title         string     `json:"title"`
	Slug          string     `json:"slug"`
	Synopsis      *string    `json:"synopsis,omitempty"`
	CoverURL      string     `json:"cover_url"`
	TrailerURL    *string    `json:"trailer_url,omitempty"`
	TotalEpisode  *string    `json:"total_episodes,omitempty"`
	Studio        *string    `json:"studio,omitempty"`
	Season        *string    `json:"season,omitempty"`
	ReleaseDate   *string    `json:"release_date,omitempty"`
	Episodes      []*Episode `json:"episodes,omitempty"`
	CacheExpireAt *time.Time `json:"cache_expire_at,omitempty"`
}

func (a *Anime) Get(db db.DBInterface) error {
	if a.ID == 0 {
		return fmt.Errorf("ID_IS_ZERO")
	}

	animeID := strconv.Itoa(a.ID)
	content, err := db.Get(data.DBAnime, &animeID)
	if err != nil {
		return err
	}

	err = json.Unmarshal(*content, a)
	if err != nil {
		return err
	}

	return nil
}

func (a *Anime) Save(db db.DBInterface, forceSave bool) error {
	if !forceSave {
		if a.CacheExpireAt == nil {
			cacheExpireAt := time.Now().Add(time.Hour * 24 * 3) // 3 days
			a.CacheExpireAt = &cacheExpireAt
		} else if a.CacheExpireAt.Before(time.Now()) {
			cacheExpireAt := time.Now().Add(time.Hour * 24 * 3) // 3 days
			a.CacheExpireAt = &cacheExpireAt
		}
	}

	_content, err := json.Marshal(a)
	if err != nil {
		return err
	}

	animeID := strconv.Itoa(a.ID)
	err = db.Save(data.DBAnime, &animeID, &_content)
	if err != nil {
		return err
	}

	return nil
}

func (a *Anime) IsDataComplete() bool {
	if a.ID == 0 {
		return false
	}

	if a.Title == "" {
		return false
	}

	if a.Slug == "" {
		return false
	}

	if a.Synopsis == nil {
		return false
	}

	if a.TrailerURL == nil {
		return false
	}

	if a.CoverURL == "" {
		return false
	}

	if a.TotalEpisode == nil {
		return false
	}

	if a.Studio == nil {
		return false
	}

	if a.Season == nil {
		return false
	}

	if a.ReleaseDate == nil {
		return false
	}

	return true
}

func (a *Anime) IsCacheExpired() bool {
	if a.CacheExpireAt == nil {
		return true
	}

	if a.CacheExpireAt.Before(time.Now()) {
		return true
	}

	return false
}

func (a *Anime) Update(db db.DBInterface, updatedAnime *Anime) error {
	if !a.IsDataComplete() || a.IsCacheExpired() {
		a.Title = updatedAnime.Title
		a.Slug = updatedAnime.Slug
		a.Synopsis = updatedAnime.Synopsis
		a.TrailerURL = updatedAnime.TrailerURL
		a.CoverURL = updatedAnime.CoverURL
		a.TotalEpisode = updatedAnime.TotalEpisode
		a.Studio = updatedAnime.Studio
		a.Season = updatedAnime.Season
		a.ReleaseDate = updatedAnime.ReleaseDate

		if updatedAnime.Episodes != nil {
			a.Episodes = updatedAnime.Episodes
		}

		err := a.Save(db, false)
		if err != nil {
			return err
		}

		return nil
	}

	return nil
}

func (a *Anime) ReOrderedEpisodes() {
	if a.Episodes == nil {
		return
	}

	var reOrderedEpisodes []*Episode
	for i := 0; i < len(a.Episodes); i++ {
		temp := a.Episodes[i]
		if i == 0 {
			reOrderedEpisodes = append(reOrderedEpisodes, temp)
			continue
		}

		for j := 0; j < len(reOrderedEpisodes); j++ {
			if temp.ID > reOrderedEpisodes[j].ID {
				reOrderedEpisodes = append(reOrderedEpisodes[:j], append([]*Episode{temp}, reOrderedEpisodes[j:]...)...)
				break
			}
		}

		if len(reOrderedEpisodes) == i {
			reOrderedEpisodes = append(reOrderedEpisodes, temp)
		}
	}

	a.Episodes = reOrderedEpisodes
}

type Episode struct {
	ID        int       `json:"id"`
	Slug      string    `json:"slug"`
	Anime     *Anime    `json:"anime,omitempty"`
	Episode   string    `json:"episode"`
	Watches   []*Watch  `json:"watches,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Watch struct {
	ID        int    `json:"id"`
	Source    string `json:"source"`
	StreamURL string `json:"stream_url"`
}
