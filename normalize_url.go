package main

import (
	"log"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

func normalizeURL(inputUrl string) (string, error) {
	u, err := url.Parse(inputUrl)
	if err != nil {
		return "", err
	}
	path := u.Path
	if path == "" {
		path = "/"
	}

	return u.Host + path, nil
}

func getURLsFromHTML(htmlBody, rawBaseURL string) ([]string, error) {
	sr := strings.NewReader(htmlBody)
	doc, err := html.Parse(sr)
	if err != nil {
		return nil, err
	}

	urls := make([]string, 0)
	addUrl := func(s string) {
		u, err := url.Parse(s)
		if err != nil {
			return
		}

		if u.IsAbs() {
			urls = append(urls, u.String())
			return
		}

		fullPath, err := url.JoinPath(rawBaseURL, u.String())
		if err != nil {
			log.Println("Invalid url:", u, err)
		}

		urls = append(urls, fullPath)
	}

	var walkNodeLevel func(*html.Node)
	walkNodeLevel = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					addUrl(a.Val)
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walkNodeLevel(c)
		}
	}
	walkNodeLevel(doc)
	// fmt.Println(urls)
	return urls, nil
}
