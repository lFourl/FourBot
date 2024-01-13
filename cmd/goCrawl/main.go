package main

import (
    "fmt"
    "net/http"
    "io/ioutil"
    "sync"
    "net" 
    "os"  
)

type Result struct {
    URL   string
    Body  string
    Error error
}

func crawl(url string, ch chan<- Result, client *http.Client) {
    resp, err := client.Get(url)
    if err != nil {
        var netErr net.Error
        if os.IsTimeout(err) {
            ch <- Result{URL: url, Error: fmt.Errorf("timeout error: %s", err)}
        } else if errors.As(err, &netErr) && netErr.Timeout() {
            ch <- Result{URL: url, Error: fmt.Errorf("network timeout error: %s", err)}
        } else if err, ok := err.(net.Error); ok && err.Timeout() {
            ch <- Result{URL: url, Error: fmt.Errorf("network timeout error: %s", err)}
        } else {
            ch <- Result{URL: url, Error: err}
        }
        return
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        ch <- Result{URL: url, Error: fmt.Errorf("status code: %d", resp.StatusCode)}
        return
    }

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        ch <- Result{URL: url, Error: err}
        return
    }

    ch <- Result{URL: url, Body: string(body)}
}

func main() {
    urls := []string{
        "http://example.com",
        "http://example.org",
        "http://example.net",
        // Add more URLs here
    }

    ch := make(chan Result)
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

    for _, url := range urls {
        wg.Add(1)
        go func(url string) {
            defer wg.Done()
            crawl(url, ch, client)
        }(url)
    }

    go func() {
        wg.Wait()
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
