package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/temoto/robotstxt"
)

type Result struct {
	URL   string
	Body  string
	Error error
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
}

func getRobotsTxt(robotsURL string, client *http.Client) (*robotstxt.RobotsData, error) {
	resp, err := client.Get(robotsURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("robots.txt not found")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	robotsData, err := robotstxt.FromBytes(body)
	if err != nil {
		return nil, err
	}

	return robotsData, nil
}

func crawl(targetURL string, ch chan<- Result, client *http.Client, rateLimiter <-chan time.Time, robotsData *robotstxt.RobotsData) {
	<-rateLimiter

	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		ch <- Result{URL: targetURL, Error: err}
		return
	}

	if robotsData != nil && !robotsData.TestAgent(parsedURL.Path, "FourBot") {
		ch <- Result{URL: targetURL, Error: fmt.Errorf("disallowed by robots.txt")}
		return
	}

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		ch <- Result{URL: targetURL, Error: err}
		return
	}

	req.Header.Set("User-Agent", "FourBot")

	resp, err := client.Do(req)
	if err != nil {
		var netErr net.Error
		if os.IsTimeout(err) {
			ch <- Result{URL: targetURL, Error: fmt.Errorf("timeout error: %s", err)}
		} else if errors.As(err, &netErr) && netErr.Timeout() {
			ch <- Result{URL: targetURL, Error: fmt.Errorf("network timeout error: %s", err)}
		} else if err, ok := err.(net.Error); ok && err.Timeout() {
			ch <- Result{URL: targetURL, Error: fmt.Errorf("network timeout error: %s", err)}
		} else {
			ch <- Result{URL: targetURL, Error: err}
		}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		ch <- Result{URL: targetURL, Error: fmt.Errorf("status code: %d", resp.StatusCode)}
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error crawling URL %s: %v\n", targetURL, err)
		ch <- Result{URL: targetURL, Error: err}
		return
	}

	ch <- Result{URL: targetURL, Body: string(body)}
}

func worker(id int, urls <-chan string, results chan<- Result, client *http.Client, rateLimiter <-chan time.Time, robotsData *robotstxt.RobotsData, wg *sync.WaitGroup) {
	log.Printf("Worker %d started\n", id)
	for url := range urls {
		log.Printf("Worker %d processing URL: %\n", id, url)
		crawl(url, results, client, rateLimiter, robotsData)
		wg.Done()
	}
	fmt.Printf("Worker %d finished\n", id)
}

func main() {
	urls := []string{
		"http://example.com",
		"http://example.org",
		"http://example.net",
		// Add more URLs here
	}

	ch := make(chan Result)
	urlsChan := make(chan string)
	var wg sync.WaitGroup

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return nil
		},
		Timeout: time.Second * 10,
	}

	rateLimit := time.Second / 10
	rateLimiter := time.Tick(rateLimit)

	const numWorkers = 5
	for i := 0; i < numWorkers; i++ {
		go worker(i, urlsChan, ch, client, rateLimiter, nil, &wg)
	}

	for _, targetURL := range urls {
		parsedURL, err := url.Parse(targetURL)
		if err != nil {
			log.Printf("Error parsing URL:", targetURL, err)
			continue
		}
		robotsURL := fmt.Sprintf("%s://%s/robots.txt", parsedURL.Scheme, parsedURL.Host)
		robotsData, err := getRobotsTxt(robotsURL, client)
		if err != nil {
			fmt.Println("Error fetching robots.txt:", err)
			continue
		}

		wg.Add(1)
		urlsChan <- targetURL
		go func(targetURL string) {
			defer wg.Done()
			crawl(targetURL, ch, client, rateLimiter, robotsData)
		}(targetURL)
	}

	go func() {
		wg.Wait()
		close(urlsChan)
		close(ch)
	}()

	for result := range ch {
		if result.Error != nil {
			fmt.Printf("Error fetching %s: %s\n", result.URL, result.Error)
		} else {
			fmt.Println("Fetched URL:", result.URL)
			fmt.Println("Content:", result.Body)
		}
	}
}
