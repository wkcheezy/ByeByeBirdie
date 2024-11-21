package main

import (
	"fmt"
	"github.com/charmbracelet/log"
	"net/http"
	"strings"
	"sync"
	"time"
)

func main() {
	log.Info("Starting up...")
	// Panic catcher
	defer func() {
		if r := recover(); r != nil {
			log.Error("Unable to complete process. Please file an issue on GitHub (https://github.com/wkcheezy/ByeByeBirdie/issues/new) or reach out on BlueSky (@saycheezy.bsky.social), and include the error message below.")
			log.Fatalf("%s", r)
		}
	}()

	log.Info("Authenticating...")
	uri, referrer, token, auth, cookies := authenticate()

	log.Info("Getting tweets...")
	tweetIds := getTweetIds(uri, referrer, token, auth, cookies)

	log.Info("Done looking for tweets.")
	if len(tweetIds) > 0 {
		var wg sync.WaitGroup
		wg.Add(len(tweetIds))
		everythingDeleted := true

		for i, tweetId := range tweetIds {
			go func() {
				defer func() {
					if r := recover(); r != nil {
						log.Error("Error deleting %s: %s", tweetId, r)
					}
					wg.Done()
					return
				}()
				payload := strings.NewReader(fmt.Sprintf(`{"variables": {"tweet_id": "%s","dark_request": false},"queryId":"VaenaVgh5q5ih7kvyVjgtg"}`, tweetId))

				req, err := http.NewRequest(http.MethodPost, "https://x.com/i/api/graphql/VaenaVgh5q5ih7kvyVjgtg/DeleteTweet", payload)

				if err != nil {
					panic(fmt.Sprintf("Error creating delete request: %s", err))
				}

				applyHeaders(req, referrer, token, auth, cookies)
				req.Header.Add("Content-Type", "application/json")

				res, err := http.DefaultClient.Do(req)
				if err != nil {
					panic(fmt.Sprintf("Error executing delete request: %s", err))
				}
				defer res.Body.Close()

				if res.StatusCode != http.StatusOK {
					if everythingDeleted == true {
						everythingDeleted = false
					}
					panic(fmt.Sprintf("Non-success response receieved: %s", res.Status))
				} else {
					log.Infof("Tweet %s deleted (%d/%d)", tweetId, i + 1, len(tweetIds))
				}
			}()
			time.Sleep(50 * time.Millisecond)
		}
		wg.Wait()
		log.Info("All done!")
		if everythingDeleted == false {
			log.Warn("There were some issues deleting some of the tweets. Please check the logs above, and reach out with any unknown issues\nGITHUB: https://github.com/wkcheezy/ByeByeBirdie/issues/new\nBLUESKY: @saycheezy.bsky.social")
		}
	} else {
		log.Error("No tweets found! If you have tweets on this account, please file an issue on GitHub: https://github.com/wkcheezy/ByeByeBirdie/issues/new or reach out on BlueSky: @saycheezy.bsky.social")
	}

}
