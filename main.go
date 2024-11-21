package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

func main() {
	uri, referrer, token, auth, cookies := authenticate()

	tweetIds := getTweetIds(uri, referrer, token, auth, cookies)

	if len(tweetIds) > 0 {
		var wg sync.WaitGroup
		wg.Add(len(tweetIds))

		const queryId = "VaenaVgh5q5ih7kvyVjgtg"
		for _, tweetId := range tweetIds {
			go func() {
				defer wg.Done()
				payload := strings.NewReader(fmt.Sprintf(`{"variables": {"tweet_id": "%s","dark_request": false},"queryId":"VaenaVgh5q5ih7kvyVjgtg"}`, tweetId))

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
