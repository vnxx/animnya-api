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
