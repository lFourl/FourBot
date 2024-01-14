package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/temoto/robotstxt"
)

func TestCrawl(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte("Hello, client"))
	}))
	defer server.Close()

	client := server.Client()
	ch := make(chan Result, 1)
	rateLimiter := time.Tick(time.Millisecond * 50)
	var robotsData *robotstxt.RobotsData

	go crawl(server.URL, ch, client, rateLimiter, robotsData)

	result := <-ch
	if result.Error != nil {
		t.Errorf("crawl() resulted in an error: %v", result.Error)
	}
	if result.Body != "Hello, client" {
		t.Errorf("crawl() body = %v, want Hello, client", result.Body)
	}
}
