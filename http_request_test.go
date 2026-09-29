package main

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// useTokenServer points GetAccessToken at handler for the duration of the
// test and restores the global token afterwards.
func useTokenServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	oldURL, oldToken := tokenURL, token
	tokenURL = srv.URL
	t.Cleanup(func() { tokenURL, token = oldURL, oldToken })
}

func TestGetAccessToken(t *testing.T) {
	tests := []struct {
		name         string
		status       int
		body         string
		want         string
		wantRejected bool
		wantErr      bool
	}{
		{"valid cookie", http.StatusOK, `{"access_token":"abc"}`, "abc", false, false},
		{"invalid cookie", http.StatusBadRequest, `{"code":"auth.obtain_access_token.oauth2_error","error":"invalid_grant"}`, "", true, true},
		{"unauthorized", http.StatusUnauthorized, `{}`, "", true, true},
		{"server error", http.StatusInternalServerError, `oops`, "", false, true},
		{"empty token", http.StatusOK, `{"access_token":""}`, "", false, true},
		{"malformed JSON", http.StatusOK, `not json`, "", false, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			useTokenServer(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			})

			got, err := GetAccessToken("cookie")
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if errors.Is(err, errAuthRejected) != tc.wantRejected {
				t.Errorf("errors.Is(err, errAuthRejected) = %v, want %v (err: %v)", !tc.wantRejected, tc.wantRejected, err)
			}
			if got != tc.want {
				t.Errorf("token = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDoRequestRefreshesTokenOnce(t *testing.T) {
	useTokenServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"access_token":"fresh"}`)
	})
	var calls atomic.Int32
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("Authorization") != "Bearer fresh" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		// Echo the body to prove the retry resent it.
		_, _ = io.Copy(w, r.Body)
	}))
	defer api.Close()

	req, err := http.NewRequest(http.MethodPost, api.URL, strings.NewReader("payload"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer stale")

	resp, err := DoRequest(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "payload" {
		t.Errorf("retried body = %q, want %q", body, "payload")
	}
	if n := calls.Load(); n != 2 {
		t.Errorf("API called %d times, want 2", n)
	}
	if token != "fresh" {
		t.Errorf("token = %q, want %q", token, "fresh")
	}
}

func TestDoRequestStopsWhenCookieRejected(t *testing.T) {
	var tokenCalls atomic.Int32
	useTokenServer(t, func(w http.ResponseWriter, r *http.Request) {
		tokenCalls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":"invalid_grant"}`)
	})
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer api.Close()

	req, err := http.NewRequest(http.MethodGet, api.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := DoRequest(req); !errors.Is(err, errAuthRejected) {
		t.Errorf("err = %v, want errAuthRejected", err)
	}
	if n := tokenCalls.Load(); n != 1 {
		t.Errorf("token endpoint called %d times, want 1", n)
	}
}

func TestDoRequestStopsWhenStillUnauthorized(t *testing.T) {
	var tokenCalls atomic.Int32
	useTokenServer(t, func(w http.ResponseWriter, r *http.Request) {
		tokenCalls.Add(1)
		_, _ = io.WriteString(w, `{"access_token":"fresh"}`)
	})
	var apiCalls atomic.Int32
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalls.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer api.Close()

	req, err := http.NewRequest(http.MethodGet, api.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := DoRequest(req); err == nil {
		t.Error("expected an error after a second 401, got nil")
	}
	if n := tokenCalls.Load(); n != 1 {
		t.Errorf("token endpoint called %d times, want 1", n)
	}
	if n := apiCalls.Load(); n != 2 {
		t.Errorf("API called %d times, want 2", n)
	}
}
