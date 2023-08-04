package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

func crawl(url string, wg *sync.WaitGroup, urlChan chan string) {
	defer wg.Done()

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error fetching URL:", url, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error: Non-OK status code for URL:", url)
		return
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		fmt.Println("Error parsing response:", url, err)
		return
	}

	visitNode(doc, urlChan)
}

func visitNode(n *html.Node, urlChan chan string) {
	if n.Type == html.ElementNode && n.Data == "a" {
		for _, attr := range n.Attr {
			if attr.Key == "href" {
				url := attr.Val
				if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
					urlChan <- url
				}
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		visitNode(c, urlChan)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <website>")
		return
	}

	website := os.Args[1]

	urlChan := make(chan string)
	var wg sync.WaitGroup

	wg.Add(1)
	go crawl(website, &wg, urlChan)

	go func() {
		for url := range urlChan {
			fmt.Println(url)
		}
	}()

	wg.Wait()

	close(urlChan)
}
