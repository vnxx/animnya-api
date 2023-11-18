package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"animenya.site/data"
	"animenya.site/model"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

func (h *Handler) LatestAnimeEpisode(c *fiber.Ctx) error {
	var result struct {
		Data  []*model.Episode `json:"data"`
		Error any              `json:"error"`
	}
	result.Data = []*model.Episode{}

	episodes, err := h.Fetcher.GetLatestAnimeEpisode(c.Context(), "")
	if err != nil {
		if err.Error() == "NOT_FOUND" {
			return c.Status(fiber.StatusOK).JSON(result)
		}

		log.Error().Err(err).Msg("anime.AllAnime: failed to get anime list")
		return c.Status(fiber.StatusInternalServerError).JSON(result)
	}
	if len(episodes) < 1 {
		return c.Status(fiber.StatusOK).JSON(result)
	}

	for _, episode := range episodes {
		animeID := strconv.Itoa(episode.Anime.ID)
		_anime, err := h.DB.Get(data.DBAnime, &animeID)
		if err != nil {
			if err.Error() != "NOT_FOUND" {
				log.Error().Err(err).Msg("anime.AllAnime: failed to get anime from db")
				continue
			}
		}

		var anime model.Anime
		if _anime != nil {
			if err := json.Unmarshal(*_anime, &anime); err != nil {
				log.Error().Err(err).Msg("anime.AllAnime: failed to unmarshal anime from db")
				continue
			}

			episode.Anime.ID = anime.ID
			episode.Anime.Title = anime.Title
			episode.Anime.Slug = anime.Slug
			episode.Anime.CoverURL = anime.CoverURL

			if anime.Episodes == nil {
				anime.Episodes = []*model.Episode{{
					ID:        episode.ID,
					Slug:      episode.Slug,
					Episode:   episode.Episode,
					CreatedAt: episode.CreatedAt,
				}}
			} else if len(anime.Episodes) > 0 {
				var found bool
				for _, _episode := range anime.Episodes {
					if _episode.ID == episode.ID {
						found = true
						break
					}
				}

				if !found {
					anime.Episodes = append(anime.Episodes, &model.Episode{
						ID:        episode.ID,
						Slug:      episode.Slug,
						Episode:   episode.Episode,
						CreatedAt: episode.CreatedAt,
					})
				}
			}
		} else {
			anime.ID = episode.Anime.ID
			anime.Title = episode.Anime.Title
			anime.Slug = episode.Anime.Slug
			anime.CoverURL = episode.Anime.CoverURL
			anime.Episodes = []*model.Episode{{
				ID:        episode.ID,
				Slug:      episode.Slug,
				Episode:   episode.Episode,
				CreatedAt: episode.CreatedAt,
			}}

			episode.Anime = &anime
		}

		err = anime.Save(h.DB, true)
		if err != nil {
			log.Error().Err(err).Msg("anime.AllAnime: failed to save anime to db")
			continue
		}

		episode.Anime.CoverURL = fmt.Sprintf(os.Getenv("API_URL")+"/anime/%d/cover", anime.ID)
	}

	result.Data = episodes
	return c.Status(fiber.StatusOK).JSON(result)
}

func (h *Handler) Anime(c *fiber.Ctx) error {
	var result struct {
		Data  *model.Anime `json:"data"`
		Error any          `json:"error"`
	}

	animeID := c.Params("anime_id")
	_anime, err := h.DB.Get(data.DBAnime, &animeID)
	if err != nil {
		if err.Error() != "NOT_FOUND" {
			log.Error().Err(err).Msg("anime.Anime: failed to get anime from db")
			return c.Status(fiber.StatusInternalServerError).JSON(result)
		}
	}
	if _anime == nil {
		result.Error = "NOT_FOUND"
		return c.Status(fiber.StatusNotFound).JSON(result)
	}

	var anime *model.Anime
	if err := json.Unmarshal(*_anime, &anime); err != nil {
		log.Error().Err(err).Msg("anime.Anime: failed to unmarshal anime from db")
		return c.Status(fiber.StatusInternalServerError).JSON(result)
	}

	if !anime.IsDataComplete() || anime.IsCacheExpired() {
		_anime, err := h.Fetcher.GetAnimeDetailByAnimeSlug(c.Context(), &anime.Slug)
		if err != nil {
			if err.Error() == "NOT_FOUND" {
				result.Error = "NOT_FOUND"
				return c.Status(fiber.StatusNotFound).JSON(result)
			}

			log.Error().Err(err).Msg("anime.Anime: failed to get anime detail")
			result.Error = "INTERNAL_SERVER_ERROR"
			return c.Status(fiber.StatusInternalServerError).JSON(result)
		}

		if _anime == nil {
			result.Error = "NOT_FOUND"
			return c.Status(fiber.StatusNotFound).JSON(result)
		}

		anime.PostID = _anime.PostID
		anime.CoverURL = _anime.CoverURL
		anime.TrailerURL = _anime.TrailerURL
		anime.TotalEpisode = _anime.TotalEpisode
		anime.Studio = _anime.Studio

		_anime, err = h.Fetcher.GetAnimeDetailByPostID(c.Context(), _anime.PostID)
		if err != nil {
			if err.Error() == "NOT_FOUND" {
				result.Error = "NOT_FOUND"
				return c.Status(fiber.StatusNotFound).JSON(result)
			}

			log.Error().Err(err).Msg("anime.Anime: failed to get anime detail")
			result.Error = "INTERNAL_SERVER_ERROR"
			return c.Status(fiber.StatusInternalServerError).JSON(result)
		}

		if err := anime.Update(h.DB, _anime); err != nil {
			log.Error().Err(err).Msg("anime.Anime: failed to update anime to db")
			result.Error = "INTERNAL_SERVER_ERROR"
			return c.Status(fiber.StatusInternalServerError).JSON(result)
		}
	}

	anime.ReOrderedEpisodes()

	anime.CoverURL = fmt.Sprintf(os.Getenv("API_URL")+"/anime/%d/cover", anime.ID)
	anime.PostID = nil
	result.Data = anime
	result.Data.CacheExpireAt = nil
	return c.Status(fiber.StatusOK).JSON(result)
}

func (h *Handler) AnimeCover(c *fiber.Ctx) error {
	c.Response().Header.Add("Cache-Time", "0")
	c.Set("Content-Type", "image/jpeg")

	animeID := c.Params("anime_id")
	_anime, err := h.DB.Get(data.DBAnime, &animeID)
	if err != nil {
		if err.Error() == "NOT_FOUND" {
			return c.Status(fiber.StatusNotFound).Send([]byte{})
		}

		log.Error().Err(err).Msg("anime.AnimeCover: failed to get anime from db")
		return c.Status(fiber.StatusInternalServerError).Send([]byte{})
	}

	var anime *model.Anime
	if _anime != nil {
		if err := json.Unmarshal(*_anime, &anime); err != nil {
			log.Error().Err(err).Msg("anime.AnimeCover: failed to unmarshal anime from db")
			return c.Status(fiber.StatusNotFound).Send([]byte{})
		}
	}

	if anime.CoverURL == "" {
		return c.Status(fiber.StatusNotFound).Send([]byte{})
	}

	req, err := http.NewRequest("GET", anime.CoverURL, nil)
	if err != nil {
		log.Error().Err(err).Msg("anime.AnimeCover: failed to create request")
		return c.Status(fiber.StatusInternalServerError).Send([]byte{})
	}

	req.Header.Set("referer", os.Getenv("SOURCE_URL"))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("anime.AnimeCover: failed to get cover")
		return c.Status(fiber.StatusInternalServerError).Send([]byte{})
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Error().Err(err).Msg("anime.AnimeCover: failed to get cover")
		return c.Status(fiber.StatusInternalServerError).Send([]byte{})
	}

	cover, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msg("anime.AnimeCover: failed to read cover")
		return c.Status(fiber.StatusInternalServerError).Send([]byte{})
	}

	return c.Status(fiber.StatusOK).Send(cover)
}

func (h *Handler) Episode(c *fiber.Ctx) error {
	var result struct {
		Data  *model.Episode `json:"data"`
		Error any            `json:"error"`
	}

	animeID, err := c.ParamsInt("anime_id")
	if err != nil {
		log.Error().Err(err).Msg("anime.Episode: failed to get anime id")
		result.Error = "INVALID_ANIME_ID"
		return c.Status(fiber.StatusBadRequest).JSON(result)
	}

	anime := model.Anime{ID: animeID}
	if err := anime.Get(h.DB); err != nil {
		if err.Error() != "NOT_FOUND" {
			log.Error().Err(err).Msg("anime.Episode: failed to get anime from db")
			result.Error = "INTERNAL_SERVER_ERROR"
			return c.Status(fiber.StatusInternalServerError).JSON(result)
		}

		result.Error = "ANIME_NOT_FOUND"
		return c.Status(fiber.StatusNotFound).JSON(result)
	}

	episodeID, err := c.ParamsInt("episode_id")
	if err != nil {
		log.Error().Err(err).Msg("anime.Episode: failed to get anime id")
		result.Error = "INVALID_EPISODE_ID"
		return c.Status(fiber.StatusBadRequest).JSON(result)
	}

	for _, episode := range anime.Episodes {
		if episode.ID == episodeID {
			result.Data = episode
			break
		}
	}

	if result.Data == nil {
		result.Error = "EPISODE_NOT_FOUND"
		return c.Status(fiber.StatusNotFound).JSON(result)
	}

	if result.Data.Watches == nil {
		watches, err := h.Fetcher.GetEpisodeWatchesByEpisodeIDAndEpisodeSlug(c.Context(), &episodeID, &result.Data.Slug)
		if err != nil {
			if err.Error() == "NOT_FOUND" {
				result.Error = "NOT_FOUND"
				return c.Status(fiber.StatusNotFound).JSON(result)
			}

			log.Error().Err(err).Msg("anime.Episode: failed to get episode watches")
			result.Error = "FAILED_TO_GET_EPISODE_WATCHES"
			return c.Status(fiber.StatusInternalServerError).JSON(result)
		}

		for _, episode := range anime.Episodes {
			if episode.ID == episodeID {
				episode.Watches = watches
				break
			}
		}

		if err := anime.Save(h.DB, true); err != nil {
			log.Error().Err(err).Msg("anime.Episode: failed to save episode")
		}
		result.Data.Watches = watches
	}

	result.Data.Anime = &model.Anime{}
	result.Data.Anime.ID = anime.ID
	result.Data.Anime.Slug = anime.Slug
	result.Data.Anime.Title = anime.Title
	result.Data.Anime.CoverURL = fmt.Sprintf(os.Getenv("API_URL")+"/anime/%d/cover", anime.ID)
	return c.Status(fiber.StatusOK).JSON(result)
}

func (h *Handler) SearchAnime(c *fiber.Ctx) error {
	c.Response().Header.Add("Cache-Time", "0")
	var result struct {
		Data  []*model.SimpleAnime `json:"data"`
		Error any                  `json:"error"`
	}
	result.Data = []*model.SimpleAnime{}

	query := c.Query("query")
	if query == "" || !(len(query) >= 3) {
		return c.Status(fiber.StatusOK).JSON(result)
	}

	anime, err := h.Fetcher.GetAnimeBySearch(c.Context(), h.DB, &query)
	if err != nil {
		log.Error().Err(err).Msg("anime.SearchAnime: failed to search anime")
		result.Error = "FAILED_TO_SEARCH_ANIME"
		return c.Status(fiber.StatusInternalServerError).JSON(result)
	}

	result.Data = anime
	return c.Status(fiber.StatusOK).JSON(result)
}
