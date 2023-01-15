package lib

import (
	"regexp"
)

func MatchStringByRegex(regex, str string) (*string, error) {
	r, err := regexp.Compile(regex)
	if err != nil {
		return nil, err
	}
	result := r.FindSubmatch([]byte(str))

	if len(result) > 1 {
		str := string(result[1])
		return &str, nil
	}

	return nil, nil
}

func MatchAllStringByRegex(regex, str string) (*[]string, error) {
	r, err := regexp.Compile(regex)
	if err != nil {
		return nil, err
	}
	result := r.FindAllSubmatch([]byte(str), -1)

	if len(result) > 0 {
		var str []string
		for _, v := range result {
			str = append(str, string(v[1]))
		}
		return &str, nil
	}

	return nil, nil
}
