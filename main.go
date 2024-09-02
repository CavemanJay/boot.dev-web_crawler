package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func getHTML(rawURL string) (string, error) {
	res, err := http.Get(rawURL)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		return "", fmt.Errorf("http error with status: %v", res.Status)
	}

	contentType := res.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		return "", fmt.Errorf("invalid content type returned")
	}

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func crawlPage(rawBaseURL, rawCurrentURL string, pages map[string]int) {
	base, err := url.Parse(rawBaseURL)
	if err != nil {
		fmt.Println("Invalid base url:", rawBaseURL, err)
		return
	}

	current, err := url.Parse(rawCurrentURL)
	if err != nil {
		fmt.Println("Invalid current url:", rawBaseURL, err)
		return
	}

	if current.Host != base.Host {
		fmt.Printf("Current (%s) is not on same domain as base (%s)\n", current.String(), base.String())
		return
	}

	u, err := normalizeURL(rawCurrentURL)
	if err != nil {
		fmt.Println("Could not normalize url:", rawCurrentURL, err)
		return
	}

	if count, ok := pages[u]; ok {
		pages[u] = count + 1
		fmt.Println("Already crawled:", u, "("+rawCurrentURL+")")
		return
	} else {
		pages[u] = 1
	}

	fmt.Println("Checking:", rawCurrentURL)
	content, err := getHTML(rawCurrentURL)
	if err != nil {
		fmt.Println("Failed to request content for:", rawCurrentURL, err)
		return
	}

	urls, err := getURLsFromHTML(content, rawBaseURL)
	if err != nil {
		fmt.Println("Could not get urls from body", content, err)
		return
	}

	for _, u := range urls {
		crawlPage(rawBaseURL, u, pages)
	}
}

func main() {
	args := os.Args
	if len(args) < 2 {
		fmt.Println("no website provided")
		os.Exit(1)
	}

	if len(args) > 2 {
		fmt.Println("too many arguments provided")
		os.Exit(1)
	}

	baseUrl := args[1]
	fmt.Println("starting crawl of:", baseUrl)

	// html, err := getHTML(baseUrl)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// fmt.Println(html)
	pages := make(map[string]int)
	crawlPage(baseUrl, baseUrl, pages)

	log.Printf("pages: %#+v\n", pages)
}
