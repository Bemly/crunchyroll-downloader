package main

import (
	"fmt"
	"net/http"
)

// DoRequest sends req and, if Crunchyroll answers 401, refreshes the access
// token and retries once. A second 401 is returned as an error rather than
// refreshing again, since it means the new token is being refused too.
func DoRequest(req *http.Request) (*http.Response, error) {
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusUnauthorized {
		return resp, err
	}

	print("Access token expired. Refetching one...\n")
	resp.Body.Close()
	newToken, err := GetAccessToken(*etpRt)
	if err != nil {
		return nil, err
	}
	token = newToken
	req.Header.Set("Authorization", "Bearer "+token)
	// The first attempt already consumed req.Body, so reset it from
	// GetBody before retrying or the retry sends an empty/truncated
	// payload (which broke license requests as "invalid license type").
	if req.GetBody != nil {
		if body, err := req.GetBody(); err == nil {
			req.Body = body
		}
	}

	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		return nil, fmt.Errorf("%s %s: still unauthorized after refreshing the access token", req.Method, req.URL)
	}
	return resp, nil
}
