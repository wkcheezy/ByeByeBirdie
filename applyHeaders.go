package main

func applyHeaders(req *http.Request, referrer, token, auth string, cookies []string) {
	req.Header.Add("Host", "x.com")
	req.Header.Add("Referrer", referrer)
	req.Header.Add("x-csrf-token", token)
	req.Header.Add("authorization", auth)
	req.Header.Add("Cookie", strings.Join(cookies, ";"))
}