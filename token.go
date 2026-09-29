package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
)

var deviceId = uuid.NewString()

// tokenURL is a variable so tests can point it at a local server.
var tokenURL = "https://www.crunchyroll.com/auth/v1/token"

// errAuthRejected marks a token request Crunchyroll refused, which almost
// always means the etp_rt cookie is wrong or expired.
var errAuthRejected = errors.New("crunchyroll rejected the etp_rt cookie")

type CrunchyrollTokenResponse struct {
	AccessToken string `json:"access_token"`
}

// GetAccessToken fetches an access token from Crunchyroll.
//
// A bad cookie gets an HTTP 400 "invalid_grant" whose JSON still parses, so
// the status and the token itself are checked; otherwise an empty token is
// returned and every later request 401s.
func GetAccessToken(etpRt string) (string, error) {
	body := url.Values{}
	body.Set("device_id", deviceId)
	body.Set("device_type", "Firefox on Linux")
	body.Set("grant_type", "etp_rt_cookie")

	req, err := http.NewRequest(http.MethodPost, tokenURL, strings.NewReader(body.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Basic bm9haWhkZXZtXzZpeWcwYThsMHE6")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:147.0) Gecko/20100101 Firefox/147.0")
	req.AddCookie(&http.Cookie{Name: "device_id", Value: deviceId})
	req.AddCookie(&http.Cookie{Name: "etp_rt", Value: etpRt})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}
	defer resp.Body.Close()

	res, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden:
		return "", fmt.Errorf("%w (HTTP %d: %s)", errAuthRejected, resp.StatusCode, strings.TrimSpace(string(res)))
	default:
		return "", fmt.Errorf("failed to get access token: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(res)))
	}

	var result CrunchyrollTokenResponse
	if err := json.Unmarshal(res, &result); err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}
	if result.AccessToken == "" {
		return "", errors.New("failed to get access token: the response had no access_token")
	}

	return result.AccessToken, nil
}
