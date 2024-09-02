package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

type config struct {
	pages              map[string]int
	baseUrl            *url.URL
	mu                 *sync.Mutex
	concurrencyControl chan struct{}
	wg                 *sync.WaitGroup
}

func newConfig(baseUrl string, concurrency int) *config {
	u, err := url.Parse(baseUrl)
	if err != nil {
		log.Fatal(err)
	}
	cfg := &config{
		baseUrl:            u,
		pages:              make(map[string]int),
		concurrencyControl: make(chan struct{}, concurrency),
		mu:                 &sync.Mutex{},
		wg:                 &sync.WaitGroup{},
	}
	cfg.wg.Add(1)
	return cfg
}

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

// func  crawlPage(rawBaseURL, rawCurrentURL string, pages map[string]int) {
func (cfg *config) crawlPage(rawCurrentURL string) {
	cfg.concurrencyControl <- struct{}{}
	defer func() {
		cfg.wg.Done()
		<-cfg.concurrencyControl
	}()

	current, err := url.Parse(rawCurrentURL)
	if err != nil {
		fmt.Println("Invalid current url:", rawCurrentURL, err)
		return
	}

	if current.Host != cfg.baseUrl.Host {
		fmt.Printf("Current (%s) is not on same domain as base (%s)\n", current.String(), cfg.baseUrl.String())
		return
	}

	u, err := normalizeURL(rawCurrentURL)
	if err != nil {
		fmt.Println("Could not normalize url:", rawCurrentURL, err)
		return
	}

	if !cfg.addPageVisit(u) {
		return
	}

	time.Sleep(500 * time.Millisecond)

	fmt.Println("Checking:", rawCurrentURL)
	content, err := getHTML(rawCurrentURL)
	if err != nil {
		fmt.Println("Failed to request content for:", rawCurrentURL, err)
		return
	}

	urls, err := getURLsFromHTML(content, cfg.baseUrl.String())
	if err != nil {
		fmt.Println("Could not get urls from body", content, err)
		return
	}

	for _, u := range urls {
		cfg.wg.Add(1)
		go cfg.crawlPage(u)
	}
}

func (cfg *config) addPageVisit(normalizedURL string) (isFirst bool) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()

	u := normalizedURL
	if count, ok := cfg.pages[u]; ok {
		cfg.pages[u] = count + 1
		fmt.Println("Already crawled:", u, "("+normalizedURL+")")
		return false
	}
	cfg.pages[u] = 1
	return true
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

	cfg := newConfig(baseUrl, 3)
	cfg.crawlPage(baseUrl)

	cfg.wg.Wait()
	log.Printf("pages: %#+v\n", cfg.pages)
}
