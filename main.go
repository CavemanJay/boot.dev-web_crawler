package main

import (
	"cmp"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
)

type config struct {
	maxPages           int
	baseUrl            *url.URL
	concurrencyControl chan struct{}
	wg                 *sync.WaitGroup

	mu    *sync.RWMutex
	pages map[string]int
}

func newConfig(baseUrl string, concurrency int, maxPages int) *config {
	u, err := url.Parse(baseUrl)
	if err != nil {
		log.Fatal(err)
	}
	cfg := &config{
		baseUrl:            u,
		pages:              make(map[string]int),
		maxPages:           maxPages,
		concurrencyControl: make(chan struct{}, concurrency),
		mu:                 &sync.RWMutex{},
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

	cfg.mu.RLock()
	if len(cfg.pages) >= cfg.maxPages {
		return
	}
	cfg.mu.RUnlock()

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

	// time.Sleep(500 * time.Millisecond)

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

func printReport(pages map[string]int, baseURL string) {
	line := "============================="
	fmt.Printf("%s\n  REPORT for %s\n%s\n", line, baseURL, line)

	type crawlResult struct {
		url   string
		count int
	}

	results := make([]crawlResult, 0, len(pages))
	for url, count := range pages {
		results = append(results, crawlResult{url: url, count: count})
	}

	slices.SortFunc(results, func(a, b crawlResult) int {
		countCmp := cmp.Compare(b.count, a.count)
		if countCmp != 0 {
			return countCmp
		}

		return cmp.Compare(a.url, b.url)
	})

	for _, result := range results {
		fmt.Printf("Found %d internal links to %s\n", result.count, result.url)
	}
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Println("no website provided")
		os.Exit(1)
	}

	if len(args) > 3 {
		fmt.Println("too many arguments provided")
		os.Exit(1)
	}

	var (
		concurrency = 2
		maxPages    = math.MaxInt
	)
	if len(args) > 1 {
		c, err := strconv.Atoi(args[1])
		if err != nil {
			log.Fatal(err)
		}
		concurrency = c
	}
	if len(args) > 2 {
		c, err := strconv.Atoi(args[2])
		if err != nil {
			log.Fatal(err)
		}
		maxPages = c
	}

	baseUrl := args[0]
	fmt.Println("starting crawl of:", baseUrl)

	cfg := newConfig(baseUrl, concurrency, maxPages)
	cfg.crawlPage(baseUrl)

	cfg.wg.Wait()

	printReport(cfg.pages, cfg.baseUrl.String())
}
