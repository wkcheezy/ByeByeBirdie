package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

func applyHeaders(req *http.Request, referrer, token, auth string, cookies []string) {
	req.Header.Add("Host", "x.com")
	req.Header.Add("Referrer", referrer)
	req.Header.Add("x-csrf-token", token)
	req.Header.Add("authorization", auth)
	req.Header.Add("Cookie", strings.Join(cookies, ";"))
}

func main() {
	token, auth, uri, referrer, cookies, tweetIds := authenticate()

	req, err := http.NewRequest(http.MethodGet, uri, nil)

	if err != nil {
		log.Fatalf("Error creating request: %s", err)
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
			log.Fatalf("Error parsing URL: %s", err)
		}
		req.URL = newUrl

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Fatalf("Error executing request: %s", err)
		}
		defer res.Body.Close()
		if res.StatusCode == http.StatusOK {
			var parsedResponse TweetsResponse

			err := json.NewDecoder(res.Body).Decode(&parsedResponse)
			if err != nil {
				log.Fatalf("Error decoding response: %s", err)
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
			log.Fatalf("Non success status returned: %s", res.Status)
		}
	}

	if len(tweetIds) > 0 {
		var wg sync.WaitGroup
		wg.Add(len(tweetIds))

		const queryId = "VaenaVgh5q5ih7kvyVjgtg"
		for _, tweetId := range tweetIds {
			go func() {
				defer wg.Done()
				payload := strings.NewReader(fmt.Sprintf(`{"variables": {"tweet_id": "%s","dark_request": false},"queryId":"VaenaVgh5q5ih7kvyVjgtg"}`, tweetId))
				if err != nil {
					log.Fatalf("Error marshalling delete body: %s", err)
				}
				uri := fmt.Sprintf("https://x.com/i/api/graphql/%s/DeleteTweet", queryId)

				req, err := http.NewRequest(http.MethodPost, uri, payload)

				if err != nil {
					log.Fatalf("Error creating delete request: %s", err)
				}

				applyHeaders(req, referrer, token, auth, cookies)
				req.Header.Add("Content-Type", "application/json")

				res, err := http.DefaultClient.Do(req)
				if err != nil {
					log.Fatalf("Error executing delete request: %s", err)
				}
				defer res.Body.Close()

				if res.StatusCode != http.StatusOK {
					body, err := io.ReadAll(res.Body)
					if err != nil {
						fmt.Println(err)
						return
					}
					fmt.Println(string(body))
					log.Fatalf("Non-success response receieved: %s", res.Status)
				}
			}()
			time.Sleep(50 * time.Millisecond)
		}
		wg.Wait()
	} else {
		log.Print("No tweets found!")
	}
}
