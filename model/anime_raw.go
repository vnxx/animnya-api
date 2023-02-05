package model

type EpisodeRaw struct {
	ID    int
	Date  string
	Slug  string
	Title struct {
		Rendered string
	}
	Categories      []int
	Yoast_Head_Json struct {
		Og_Image []struct {
			URL string
		}
	}
}

type AnimeDetailRaw struct {
	Title    string
	Cover    string `json:"img"`
	Duration string
	Released string
	Status   string
	Score    string
	Genre    []struct {
		Name string
		Slug string `json:"link"`
	}
	Season []struct {
		Name string
		Slug string `json:"link"`
	}
	Synopsis string
	Data     []struct {
		Episode string
		URL     string
		Player  []struct {
			Title string
			URL   string
		}
	}
}
