package lib

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"animenya.site/model"
	"github.com/rs/zerolog/log"
)

type FetcherInterface interface {
	Do(context.Context, string, string, interface{}, io.Reader, *map[string]string) (*string, error)
	GetLatestAnimeEpisode(ctx context.Context, params string) ([]*model.Episode, error)
	GetAllEpisodesByAnimeID(ctx context.Context, animeID string) ([]*model.Episode, error)
	GetAnimeDetailByAnimeSlug(ctx context.Context, animeSlug *string) (*model.Anime, error)
	GetEpisodeWatchesByEpisodeIDAndEpisodeSlug(ctx context.Context, episodeID *int, episodeSlug *string) ([]*model.Watch, error)
}

func NewFetcher() *Fetcher {
	return &Fetcher{}
}

type Fetcher struct{}

func (f *Fetcher) getAnimeEpisode(ctx context.Context, endpoint *string) ([]*model.Episode, error) {
	if endpoint == nil {
		return nil, nil
	}

	var resp []*model.EpisodeRaw
	_, err := f.Do(ctx, *endpoint, http.MethodGet, &resp, nil, nil)
	if err != nil {
		return nil, err
	}

	var result []*model.Episode
	for _, item := range resp {
		var coverURL string
		if len(item.Yoast_Head_Json.Og_Image) > 0 {
			coverURL = item.Yoast_Head_Json.Og_Image[0].URL
		}

		var categoryID int
		if len(item.Categories) > 0 {
			categoryID = item.Categories[0]
		}

		title, err := MatchStringByRegex(`(.*).(?:Episode.*)`, item.Title.Rendered)
		if err != nil {
			log.Error().Err(err).Msg("fetcher.GetLatestAnimeEpisode: failed to parse title")
			return nil, err
		}

		episode, err := MatchStringByRegex(`(?:Episode.)(.*)`, item.Title.Rendered)
		if err != nil {
			log.Error().Err(err).Msg("fetcher.GetLatestAnimeEpisode: failed to parse episode")
			return nil, err
		}

		slug, err := MatchStringByRegex(`(.*)-(?:episode.*)`, item.Slug)
		if err != nil {
			log.Error().Err(err).Msg("fetcher.GetLatestAnimeEpisode: failed to parse slug")
			return nil, err
		}

		date, err := time.Parse("2006-01-02T15:04:05", item.Date)
		if err != nil {
			log.Error().Err(err).Msg("fetcher.GetLatestAnimeEpisode: failed to parse date")
			return nil, err
		}

		result = append(result, &model.Episode{
			ID:      item.ID,
			Episode: *episode,
			Slug:    item.Slug,
			Anime: &model.Anime{
				ID:       categoryID,
				Title:    *title,
				Slug:     *slug,
				CoverURL: coverURL,
			},
			CreatedAt: date,
		})
	}

	return result, nil
}

func (f *Fetcher) GetLatestAnimeEpisode(ctx context.Context, params string) ([]*model.Episode, error) {
	endpoint := "https://samehadaku.win/wp-json/wp/v2/posts?_fields=id,title,date,slug,categories,yoast_head_json.og_image&per_page=20&" + params
	results, err := f.getAnimeEpisode(ctx, &endpoint)
	if err != nil {
		return nil, err
	}

	return results, nil
}

func (f *Fetcher) GetAllEpisodesByAnimeID(ctx context.Context, animeID string) ([]*model.Episode, error) {
	endpoint := "https://samehadaku.win/wp-json/wp/v2/posts?_fields=id,title,date,slug,categories,yoast_head_json.og_image&per_page=100&categories=" + animeID
	results, err := f.getAnimeEpisode(ctx, &endpoint)
	if err != nil {
		return nil, err
	}

	return results, nil
}

func (f *Fetcher) GetAnimeDetailByAnimeSlug(ctx context.Context, animeSlug *string) (*model.Anime, error) {
	if animeSlug == nil {
		return nil, nil
	}

	endpoint := "https://samehadaku.win/anime/" + *animeSlug
	body, err := f.Do(ctx, endpoint, http.MethodGet, nil, nil, nil)
	if err != nil {
		return nil, err
	}

	var anime model.Anime
	anime.Slug = *animeSlug

	coverURL, err := MatchStringByRegex(`.*c="(.*)" class="anmsa`, *body)
	if err != nil {
		log.Error().Err(err).Msg("fetcher.GetAnimeDetailByAnimeSlug: failed to parse cover url")
	}
	if coverURL != nil {
		anime.CoverURL = *coverURL
	}

	title, err := MatchStringByRegex(`class="anmsa" title="(.*)" alt.*rt"`, *body)
	if err != nil {
		log.Error().Err(err).Msg("fetcher.GetAnimeDetailByAnimeSlug: failed to parse title")
	}
	if title != nil {
		anime.Title = *title
	}

	synopsis, err := MatchStringByRegex(`description".(.*)</div></div><div class="genre-info">`, *body)
	if err != nil {
		log.Error().Err(err).Msg("fetcher.GetAnimeDetailByAnimeSlug: failed to parse synopsis")
	}
	if synopsis != nil {
		anime.Synopsis = synopsis
	}

	trailerURL, err := MatchStringByRegex(`player-embed.*src="(.*)">"`, *body)
	if err != nil {
		log.Error().Err(err).Msg("fetcher.GetAnimeDetailByAnimeSlug: failed to parse trailer url")
	}
	if trailerURL != nil {
		anime.TrailerURL = trailerURL
	}

	totalEpisode, err := MatchStringByRegex(`Total Episode.*>.(.*)<`, *body)
	if err != nil {
		log.Error().Err(err).Msg("fetcher.GetAnimeDetailByAnimeSlug: failed to parse total episode")
	}
	if totalEpisode != nil {
		anime.TotalEpisode = totalEpisode
	}

	studio, err := MatchStringByRegex(`Studio.*"tag".(.*)</a`, *body)
	if err != nil {
		log.Error().Err(err).Msg("fetcher.GetAnimeDetailByAnimeSlug: failed to parse studio")
	}
	if studio != nil {
		anime.Studio = studio
	}

	season, err := MatchStringByRegex(`Season..*season.*">(.*)</a></span>.*<span><b>Studio</b>`, *body)
	if err != nil {
		log.Error().Err(err).Msg("fetcher.GetAnimeDetailByAnimeSlug: failed to parse season")
	}
	if season != nil {
		anime.Season = season
	}

	releaseDate, err := MatchStringByRegex(`Rilis:.*b>(.*)</span`, *body)
	if err != nil {
		log.Error().Err(err).Msg("fetcher.GetAnimeDetailByAnimeSlug: failed to parse release date")
	}
	if releaseDate != nil {
		anime.ReleaseDate = releaseDate
	}

	return &anime, nil
}

func (f *Fetcher) GetEpisodeWatchesByEpisodeIDAndEpisodeSlug(ctx context.Context, episodeID *int, episodeSlug *string) ([]*model.Watch, error) {
	if episodeSlug == nil {
		return nil, fmt.Errorf("EPISODE_SLUG_NOT_FOUND")
	}

	if episodeID == nil {
		return nil, fmt.Errorf("EPISODE_ID_NOT_FOUND")
	}

	endpoint := "https://samehadaku.win/" + *episodeSlug
	body, err := f.Do(ctx, endpoint, http.MethodGet, nil, nil, nil)
	if err != nil {
		return nil, err
	}

	_watch, err := MatchAllStringByRegex(`data-nume=".*<span>(.*)</span`, *body)
	if err != nil {
		log.Error().Err(err).Msg("fetcher.GetEpisodeDetailByEpisodeSlug: failed to parse watch")
		return nil, err
	}
	if _watch == nil {
		return nil, fmt.Errorf("WATCH_NOT_FOUND")
	}

	var watches []*model.Watch
	headers := map[string]string{"Content-Type": "application/x-www-form-urlencoded"}
	for i, w := range *_watch {
		param := url.Values{}
		param.Set("action", "player_ajax")
		param.Set("post", strconv.Itoa(*episodeID))
		param.Set("nume", strconv.Itoa(i+1))
		param.Set("type", "schtml")
		playload := bytes.NewBufferString(param.Encode())
		body, err = f.Do(ctx, "https://samehadaku.win/wp-admin/admin-ajax.php", http.MethodPost, nil, playload, &headers)
		if err != nil {
			log.Error().Err(err).Msg("fetcher.GetEpisodeDetailByEpisodeSlug: failed to fetch watch")
			continue
		}

		if body == nil {
			log.Error().Err(err).Msg("fetcher.GetEpisodeDetailByEpisodeSlug: failed to fetch watch")
			continue
		}

		streamURL, err := MatchStringByRegex(`src="(.*)".F`, *body)
		if err != nil {
			log.Error().Err(err).Msg("fetcher.GetEpisodeDetailByEpisodeSlug: failed to parse streamURL")
			continue
		}
		if streamURL == nil {
			log.Error().Err(err).Msg("fetcher.GetEpisodeDetailByEpisodeSlug: failed to parse streamURL")
			continue
		}

		watches = append(watches, &model.Watch{
			ID:        i + 1,
			Source:    w,
			StreamURL: *streamURL,
		})
	}

	if len(watches) == 0 {
		return nil, fmt.Errorf("NO_WATCH_FOUND")
	}

	return watches, nil
}

func (f *Fetcher) Do(ctx context.Context, url string, method string, target interface{}, body io.Reader, headers *map[string]string) (*string, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		log.Error().Err(err).Msg("fetcher.Do: failed to create request")
		return nil, err
	}

	if headers != nil {
		for k, v := range *headers {
			req.Header.Set(k, v)
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("fetcher.Do: failed to do request")
		return nil, err
	}
	defer resp.Body.Close()

	if target != nil {
		if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
			log.Error().Err(err).Msg("fetcher.Do: failed to decode response body")
			return nil, err
		}
		return nil, nil
	}

	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msg("fetcher.Do: failed to read response body")
		return nil, err
	}

	resBodyStr := string(resBody)
	if resBodyStr == "" {
		return nil, fmt.Errorf("EMPTY_RESPONSE_BODY")
	}
	return &resBodyStr, nil
}
