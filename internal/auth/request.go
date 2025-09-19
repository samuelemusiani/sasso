package auth

import "net/http"

func AddAuthToRequest(req *http.Request, secret string) *http.Request {
	req.Header.Set("Authorization", secret)
	return req
}
