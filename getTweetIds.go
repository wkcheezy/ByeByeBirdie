package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

func getTweetIds(uri, referrer, token, auth string, cookies []string) []string{
	var tweetIds []string
	req, err := http.NewRequest(http.MethodGet, uri, nil)

	if err != nil {
		panic(fmt.Sprintf("Error creating request: %s", err))
	}

	applyHeaders(req, referrer, token, auth, cookies)

	var prevEndCursor string
	tweetRegex := regexp.MustCompile(`^tweet-\d+$`)
	cursorRegex := regexp.MustCompile(`^cursor-bottom-\d+$`)
	keepRequesting := true

	for keepRequesting {
		keepRequesting = false

		newUrl, err := url.Parse(uri)
		if err != nil {
			panic(fmt.Sprintf("Error parsing URL: %s", err))
		}
		req.URL = newUrl

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			panic(fmt.Sprintf("Error executing request: %s", err))
		}
		defer res.Body.Close()
		if res.StatusCode == http.StatusOK {
			var parsedResponse TweetsResponse

			err := json.NewDecoder(res.Body).Decode(&parsedResponse)
			if err != nil {
				panic(fmt.Sprintf("Error decoding response: %s", err))
			}
			for _, item := range parsedResponse.Data.User.Result.Timeline_v2.Timeline.Instructions {
				for _, entry := range item.Entries {
					if tweetRegex.MatchString(entry.EntryId) {
						if entry.Content.ItemContent.Tweet_results.Result.Legacy.Retweeted_status_result == nil {
							tweetIds = append(tweetIds, entry.Content.ItemContent.Tweet_results.Result.RestID)
						}
					} else if cursorRegex.MatchString(entry.EntryId) && entry.Content.Value != prevEndCursor {
						if strings.Contains(uri, "cursor") {
							strings.Replace(uri, prevEndCursor, entry.Content.Value, 1)
						} else {
							cursor := url.QueryEscape(fmt.Sprintf("\"cursor\":\"%s\",", entry.Content.Value))
							index := strings.Index(uri, "%7B") + 3
							uri = uri[:index] + cursor + uri[index:]
						}
						prevEndCursor = entry.Content.Value
						keepRequesting = true
					}
				}
			}
		} else {
			panic(fmt.Sprintf("Non success status returned: %s", res.Status))
		}
	}

	return tweetIds
}
