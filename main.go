package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/gson"
)

func applyHeaders(req *http.Request, referrer, token, auth string, cookies []string) {
	req.Header.Add("Host", "x.com")
	req.Header.Add("Referrer", referrer)
	req.Header.Add("x-csrf-token", token)
	req.Header.Add("authorization", auth)
	req.Header.Add("Cookie", strings.Join(cookies, ";"))
}

func main() {
	if path, exists := launcher.LookPath(); exists {
		args := launcher.New().Headless(false).FormatArgs()

		var cmd *exec.Cmd
		cmd = exec.Command(path, args...)

		parser := launcher.NewURLParser()
		cmd.Stderr = parser
		utils.E(cmd.Start())
		u := launcher.MustResolveURL(<-parser.URL)

		browser := rod.New().ControlURL(u).MustConnect()
		page := browser.MustPage("https://x.com/login")

		err := page.WaitElementsMoreThan("[aria-label='Profile']", 0)
		if err != nil {
			log.Fatalf("Error waiting for elements: %s", err)
		}

		page.MustElement("[aria-label='Profile']").MustClick()

		var token, auth, uri, referrer string
		var cookies, tweetIds []string

		page.EachEvent(func(e *proto.NetworkRequestWillBeSent) (stop bool) {
			if strings.Contains(e.Request.URL, "UserTweets") && !gson.JSON.Nil(e.Request.Headers["authorization"]) {
				c, err := page.Cookies([]string{})
				if err != nil {
					log.Fatalf("Error getting cookies: %s", err)
				}
				token = e.Request.Headers["x-csrf-token"].String()
				auth = e.Request.Headers["authorization"].String()
				referrer = e.Request.Headers["Referrer"].String()
				for _, cookie := range c {
					cookies = append(cookies, fmt.Sprintf("%s=%s", cookie.Name, cookie.Value))
				}

				uri = e.Request.URL

				return true
			}
			return false
		}, func(e *proto.NetworkResponseReceived) (stop bool) {
			if strings.Contains(e.Response.URL, "UserTweets") {
				return true
			}
			return false
		})()
		browser.Close()

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

		var wg sync.WaitGroup
		wg.Add(len(tweetIds))
		for _, tweetId := range tweetIds {
			go func() {
				defer wg.Done()
				queryId := "VaenaVgh5q5ih7kvyVjgtg"
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
		log.Fatal("Missing Chromium browser")
	}
}
