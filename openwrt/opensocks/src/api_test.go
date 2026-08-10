package main

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestAuthenticatedRequestRetriesTransient20001(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) < 3 {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"fail","code":20001,"error":"auth service failed"}`))
			return
		}
		_, _ = w.Write([]byte(`{"lines":[]}`))
	}))
	defer server.Close()

	client := newAPIClient(server.URL)
	var delays []time.Duration
	client.sleep = func(delay time.Duration) { delays = append(delays, delay) }

	if _, err := client.getLines(); err != nil {
		t.Fatalf("getLines returned error after retry: %v", err)
	}
	if got := requests.Load(); got != 3 {
		t.Fatalf("request count = %d, want 3", got)
	}
	if len(delays) != 2 || delays[0] != time.Second || delays[1] != 2*time.Second {
		t.Fatalf("retry delays = %v, want [1s 2s]", delays)
	}
}

func TestAuthenticatedRequestDoesNotRetryOtherErrors(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_, _ = w.Write([]byte(`{"state":{"Success":false,"Code":20002,"Error":"bad parameter"}}`))
	}))
	defer server.Close()

	client := newAPIClient(server.URL)
	client.sleep = func(time.Duration) { t.Fatal("unexpected retry") }

	_, err := client.getLines()
	if !isAPIErrorCode(err, 20002) {
		t.Fatalf("getLines error = %v, want API code 20002", err)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("request count = %d, want 1", got)
	}
}

func TestAuthenticatedRequestStopsAfterThreeAttempts(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_, _ = w.Write([]byte(`{"status":"fail","code":20001,"error":"auth service failed"}`))
	}))
	defer server.Close()

	client := newAPIClient(server.URL)
	client.sleep = func(time.Duration) {}

	_, err := client.getLines()
	if !isAPIErrorCode(err, 20001) {
		t.Fatalf("getLines error = %v, want API code 20001", err)
	}
	if got := requests.Load(); got != 3 {
		t.Fatalf("request count = %d, want 3", got)
	}
}
