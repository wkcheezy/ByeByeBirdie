package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/gson"
)

func authenticate() (string, string, string, string, []string) {
	path, exists := launcher.LookPath()
	if !exists {
		panic(fmt.Sprintf("Missing Chromium browser"))		
	}
	args := launcher.New().Headless(false).FormatArgs()
	var cmd *exec.Cmd
	cmd = exec.Command(path, args...)

	parser := launcher.NewURLParser()
	cmd.Stderr = parser
	utils.E(cmd.Start())
	u := launcher.MustResolveURL(<-parser.URL)

	log.Info("Opening browser. Please login.")
	browser := rod.New().ControlURL(u).MustConnect()
	page := browser.MustPage("https://x.com/login")

	err := page.WaitElementsMoreThan("[aria-label='Profile']", 0)
	if err != nil {
		panic(fmt.Sprintf("Error waiting for elements: %s", err))		
	}
	log.Info("Login successful. Fetching auth headers...")

	page.MustElement("[aria-label='Profile']").MustClick()

	var uri, referrer, token, auth string
	var cookies []string

	page.EachEvent(func(e *proto.NetworkRequestWillBeSent) (stop bool) {
		if strings.Contains(e.Request.URL, "UserTweets") && !gson.JSON.Nil(e.Request.Headers["authorization"]) {
			c, err := page.Cookies([]string{})
			if err != nil {
				panic(fmt.Sprintf("Error getting cookies: %s", err))				
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
	log.Info("Authentication successful")
	return uri, referrer, token, auth, cookies
}
