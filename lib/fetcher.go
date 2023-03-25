package lib

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"animenya.site/model"
	"github.com/rs/zerolog/log"
)

type FetcherInterface interface {
	Do(context.Context, string, string, interface{}, io.Reader, *map[string]string) (*string, error)
	GetLatestAnimeEpisode(ctx context.Context, params string) ([]*model.Episode, error)
	GetAllEpisodesByAnimeID(ctx context.Context, animeID string) ([]*model.Episode, error) // deprecated: use GetAnimeDetailByPostID instead
	GetAnimeDetailByAnimeSlug(ctx context.Context, animeSlug *string) (*model.Anime, error)
	GetAnimeDetailByPostID(ctx context.Context, postID *int) (*model.Anime, error)
	// GetAnimePostIDByAnimeSlug(ctx context.Context, animeSlug *string) (*int, error)
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
			log.Error().Err(err).Msg("fetcher.getAnimeEpisode: failed to parse title")
			return nil, err
		}

		episode, err := MatchStringByRegex(`(?:Episode.)(.*)`, item.Title.Rendered)
		if err != nil {
			log.Error().Err(err).Msg("fetcher.getAnimeEpisode: failed to parse episode")
			return nil, err
		}
    // probably this is a movie: title-movie
    if episode == nil {
      continue
    }

		slug, err := MatchStringByRegex(`(.*)-(?:episode.*)`, item.Slug)
		if err != nil {
			log.Error().Err(err).Msg("fetcher.getAnimeEpisode: failed to parse slug")
			return nil, err
		}

		date, err := time.Parse("2006-01-02T15:04:05", item.Date)
		if err != nil {
			log.Error().Err(err).Msg("fetcher.getAnimeEpisode: failed to parse date")
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
			CreatedAt: &date,
		})
	}

	return result, nil
}

func (f *Fetcher) GetLatestAnimeEpisode(ctx context.Context, params string) ([]*model.Episode, error) {
	endpoint := fmt.Sprintf("%s/wp-json/wp/v2/posts?_fields=id,title,date,slug,categories,yoast_head_json.og_image&per_page=20&status=publish&%s", os.Getenv("SOURCE_URL"), params)
	results, err := f.getAnimeEpisode(ctx, &endpoint)
	if err != nil {
		return nil, err
	}

	return results, nil
}

func (f *Fetcher) GetAllEpisodesByAnimeID(ctx context.Context, animeID string) ([]*model.Episode, error) {
	endpoint := fmt.Sprintf("%s/wp-json/wp/v2/posts?_fields=id,title,date,slug,categories,yoast_head_json.og_image&per_page=100&categories=%s", os.Getenv("SOURCE_URL"), animeID)
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

	endpoint := fmt.Sprintf("%s/anime/%s", os.Getenv("SOURCE_URL"), *animeSlug)
	body, err := f.Do(ctx, endpoint, http.MethodGet, nil, nil, nil)
	if err != nil {
		return nil, err
	}

	var anime model.Anime
	anime.Slug = *animeSlug

	_postID, err := MatchStringByRegex(`id="post-(.*)" clas`, *body)
	if err != nil {
		log.Error().Err(err).Msg("fetcher.GetAnimeDetailByAnimeSlug: failed to parse post id")
		return nil, err
	}

	postID, err := strconv.Atoi(*_postID)
	if err != nil {
		log.Error().Err(err).Msg("fetcher.GetAnimeDetailByAnimeSlug: failed to parse post id")
		return nil, err
	}

	if anime.PostID == nil {
		anime.PostID = &postID
	}

	// coverURL, err := MatchStringByRegex(`.*c="(.*)" class="anmsa`, *body)
	// if err != nil {
	// 	log.Error().Err(err).Msg("fetcher.GetAnimeDetailByAnimeSlug: failed to parse cover url")
	// }
	// if coverURL != nil {
	// 	anime.CoverURL = *coverURL
	// }

	// title, err := MatchStringByRegex(`class="anmsa" title="(.*)" alt.*rt"`, *body)
	// if err != nil {
	// 	log.Error().Err(err).Msg("fetcher.GetAnimeDetailByAnimeSlug: failed to parse title")
	// }
	// if title != nil {
	// 	anime.Title = *title
	// }

	// synopsis, err := MatchStringByRegex(`description".(.*)</div></div><div class="genre-info">`, *body)
	// if err != nil {
	// 	log.Error().Err(err).Msg("fetcher.GetAnimeDetailByAnimeSlug: failed to parse synopsis")
	// }
	// if synopsis != nil {
	// 	anime.Synopsis = synopsis
	// }

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

	// season, err := MatchStringByRegex(`Season..*season.*">(.*)</a></span>.*<span><b>Studio</b>`, *body)
	// if err != nil {
	// 	log.Error().Err(err).Msg("fetcher.GetAnimeDetailByAnimeSlug: failed to parse season")
	// }
	// if season != nil {
	// 	anime.Season = season
	// }

	// releaseDate, err := MatchStringByRegex(`Rilis:.*b>(.*)</span`, *body)
	// if err != nil {
	// 	log.Error().Err(err).Msg("fetcher.GetAnimeDetailByAnimeSlug: failed to parse release date")
	// }
	// if releaseDate != nil {
	// 	anime.ReleaseDate = releaseDate
	// }

	return &anime, nil
}

func (f *Fetcher) GetAnimeDetailByPostID(ctx context.Context, postID *int) (*model.Anime, error) {
	if postID == nil {
		return nil, nil
	}

	endpoint := fmt.Sprintf(`%s/wp-json/apk/anime/?id=%d`, os.Getenv("SOURCE_URL"), *postID)
	var _animeRaw []*model.AnimeDetailRaw
	_, err := f.Do(ctx, endpoint, http.MethodGet, &_animeRaw, nil, nil)
	if err != nil {
		return nil, err
	}

	if _animeRaw == nil {
		return nil, nil
	}

	animeRaw := (_animeRaw)[0]
	var anime model.Anime
	anime.Title = animeRaw.Title
	anime.CoverURL = animeRaw.Cover
	anime.Duration = &animeRaw.Duration
	anime.Synopsis = &animeRaw.Synopsis
	anime.ReleaseDate = &animeRaw.Released
	anime.Status = &animeRaw.Status
	anime.Score = &animeRaw.Score

	for _, _genre := range animeRaw.Genre {
		var genre model.Genre
		genre.Name = _genre.Name

		slug, err := MatchStringByRegex(`&val=(.*)`, _genre.Slug)
		if err != nil {
			log.Error().Err(err).Msg("fetcher.GetAnimeDetailByPostID: failed to parse genre slug")
			continue
		}

		genre.Slug = *slug
		if anime.Genre == nil {
			anime.Genre = &[]model.Genre{}
		}

		*anime.Genre = append(*anime.Genre, genre)
	}

	for _, episodeRaw := range animeRaw.Data {
		var episode model.Episode
		episode.Episode = episodeRaw.Episode

		_id, err := MatchStringByRegex(`&id=(.*)`, episodeRaw.URL)
		if err != nil {
			log.Error().Err(err).Msg("fetcher.GetAnimeDetailByPostID: failed to parse episode id")
			continue
		}
		if _id == nil {
			log.Error().Err(err).Msg("fetcher.GetAnimeDetailByPostID: failed to parse episode id")
			continue
		}

		id, err := strconv.Atoi(*_id)
		if err != nil {
			log.Error().Err(err).Msg("fetcher.GetAnimeDetailByPostID: failed to parse episode id")
			continue
		}
		episode.ID = id

		for i, player := range episodeRaw.Player {
			var watch model.Watch
			streamURL, err := MatchStringByRegex(`src.+"(.*)".F`, player.URL)

			if err != nil {
				log.Error().Err(err).Msg("fetcher.GetEpisodeDetailByEpisodeSlug: failed to parse streamURL")
				continue
			}
			if streamURL == nil {
				log.Error().Err(err).Msg("fetcher.GetEpisodeDetailByEpisodeSlug: failed to parse streamURL")
				continue
			}

			watch.ID = i + 1
			watch.Source = player.Title
			watch.StreamURL = *streamURL
			episode.Watches = append(episode.Watches, &watch)
		}

		anime.Episodes = append(anime.Episodes, &episode)
	}

	return &anime, nil
}

// func (f *Fetcher) GetAnimePostIDByAnimeSlug(ctx context.Context, animeSlug *string) (*int, error) {
// 	if animeSlug == nil {
// 		return nil, nil
// 	}

// 	endpoint := "https://samehadaku.run/anime/" + *animeSlug
// 	body, err := f.Do(ctx, endpoint, http.MethodGet, nil, nil, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	_postID, err := MatchStringByRegex(`id="post-(.*)" clas`, *body)
// 	if err != nil {
// 		log.Error().Err(err).Msg("fetcher.GetAnimeDetailByAnimeSlug: failed to parse post id")
// 		return nil, err
// 	}

// 	postID, err := strconv.Atoi(*_postID)
// 	if err != nil {
// 		log.Error().Err(err).Msg("fetcher.GetAnimeDetailByAnimeSlug: failed to parse post id")
// 		return nil, err
// 	}

// 	return &postID, nil
// }

func (f *Fetcher) GetEpisodeWatchesByEpisodeIDAndEpisodeSlug(ctx context.Context, episodeID *int, episodeSlug *string) ([]*model.Watch, error) {
	if episodeSlug == nil {
		return nil, fmt.Errorf("EPISODE_SLUG_NOT_FOUND")
	}

	if episodeID == nil {
		return nil, fmt.Errorf("EPISODE_ID_NOT_FOUND")
	}

	endpoint := fmt.Sprintf("%s/%s", os.Getenv("SOURCE_URL") , *episodeSlug)
	body, err := f.Do(ctx, endpoint, http.MethodGet, nil, nil, nil)
	if err != nil {
		return nil, err
	}

	_watch, err := MatchAllStringByRegex(`data-nume=".*<span>(.*)</span`, *body)
	if err != nil {
		log.Error().Err(err).Msg("fetcher.GetEpisodeWatchesByEpisodeIDAndEpisodeSlug: failed to parse watch")
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
		body, err = f.Do(ctx, fmt.Sprintf("%s/wp-admin/admin-ajax.php", os.Getenv("SOURCE_URL")), http.MethodPost, nil, playload, &headers)
		if err != nil {
			log.Error().Err(err).Msg("fetcher.GetEpisodeWatchesByEpisodeIDAndEpisodeSlug: failed to fetch watch")
			continue
		}

		if body == nil {
			log.Error().Err(err).Msg("fetcher.GetEpisodeWatchesByEpisodeIDAndEpisodeSlug: failed to fetch watch")
			continue
		}

		streamURL, err := MatchStringByRegex(`src="(.*)".F`, *body)
		if err != nil {
			log.Error().Err(err).Msg("fetcher.GetEpisodeWatchesByEpisodeIDAndEpisodeSlug: failed to parse streamURL")
			continue
		}
		if streamURL == nil {
      log.Error().Err(err).Msg("fetcher.GetEpisodeWatchesByEpisodeIDAndEpisodeSlug: failed to parse streamURL")
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

// func (f *Fetcher) SearchAnime(ctx context.Context, query *string) ([]*model.Anime, error) {
// 	if query == nil {
// 		return nil, nil
// 	}

// 	endpoint := "https://samehadaku.run/?s=" + *query
// }

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

	if resp.StatusCode != http.StatusOK {
    log.Error().Err(err).Msg("fetcher.Do: failed to do request, respose status code: " + fmt.Sprintf("%d", resp.StatusCode)) 
		return nil, fmt.Errorf("STATUS_CODE_NOT_OK")
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("NOT_FOUND")
	}

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
