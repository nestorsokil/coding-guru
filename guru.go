package main

import (
	"strings"
	"net/http"
	"net/url"
	"golang.org/x/net/html"
	"io"
	"github.com/PuerkitoBio/goquery"
	"fmt"
	"context"
	"log"
)

const (
	NoResults = "Could not find what you were looking for."
	GoogleSearchString = "https://google.com/search?q=site:stackoverflow.com%20"
	CacheTTLSeconds = 3600
	NoResultsCacheTTLSeconds = 1800
)

type CodeGuru interface {
	FindAnswer(ctx context.Context, question string) (answer string, err error)
}

func NewGuru() CodeGuru {
	return &WebCrawlerCodeGuru{questionCache:NewCache(1000), webLinkCache:NewCache(1000)}
}

type WebCrawlerCodeGuru struct {
	questionCache QueryCache
	webLinkCache QueryCache
}

func (g *WebCrawlerCodeGuru) FindAnswer(ctx context.Context, question string) (answer string, err error) {
	results, errors := g.answerAsync(question)
	select {
	case answer := <- results:
		return answer, nil
	case err := <- errors:
		return "", err
	case <- ctx.Done():
		fmt.Printf("[WARN] Context was cancelled for query = '%s'", question)
		return "", ctx.Err()
	}
}

func (g *WebCrawlerCodeGuru) answerAsync(question string) (<-chan string, <-chan error) {
	resultChan := make(chan string, 1)
	errChan := make(chan error, 1)
	go func() {
		defer close(resultChan)
		defer close(errChan)

		cached, hit := g.questionCache.Get(question)
		if hit {
			log.Printf("[INFO] Hit questions cache for '%s'", question)
			resultChan <- cached
			return
		}

		searchLink := fmtSearchString(question)
		searchResp, err := http.Get(searchLink)
		if err != nil {
			errChan <- err
			return
		}
		defer searchResp.Body.Close()
		links := parseQuestionLinks(searchResp.Body, 1)
		if len(links) == 0 {
			resultChan <- NoResults
			g.questionCache.Put(question, NoResults, NoResultsCacheTTLSeconds)
			return
		}
		link := links[0]
		cached, hit = g.webLinkCache.Get(link)
		if hit {
			log.Printf("[INFO] Hit link cache for '%s'", link)
			resultChan <- cached
			g.questionCache.Put(question, cached, CacheTTLSeconds)
			return
		}

		questionResp, err := http.Get(link + "?answertab=votes")
		if err != nil {
			errChan <- err
			return
		}
		defer questionResp.Body.Close()
		answer, err := parseAnswers(questionResp.Body)
		if err != nil {
			errChan <- err
			return
		}
		if answer == "" {
			resultChan <- NoResults
			g.questionCache.Put(question, NoResults, NoResultsCacheTTLSeconds)
			g.questionCache.Put(link, NoResults, NoResultsCacheTTLSeconds)
			return
		}
		resultChan <- answer
		g.questionCache.Put(question, answer, CacheTTLSeconds)
		g.webLinkCache.Put(link, answer, CacheTTLSeconds)
	}()

	return resultChan, errChan
}

func fmtSearchString(query string) string {
	return GoogleSearchString + url.QueryEscape(query)
}

// TODO rewrite using goquery
func parseQuestionLinks(htmlReader io.Reader, maxResults int) []string {
	tokenizer := html.NewTokenizer(htmlReader)
	var links []string
	var found = 0
	for {
		if found >= maxResults {
			return links
		}
		tokenType := tokenizer.Next()
		switch tokenType {
		case html.ErrorToken:
			return links
		case html.StartTagToken:
			token := tokenizer.Token()
			if token.Data == "a" {
				ok, href := getHref(token)
				if ok && strings.Contains(href, "/questions/") {
					href = strings.Replace(href, "/url?q=", "", 1)
					link := strings.Split(href, "&sa")[0]
					links = append(links, link)
					found = found + 1
				}
			}
		}
	}
}

func getHref(t html.Token) (ok bool, href string) {
	for _, a := range t.Attr {
		if a.Key == "class" {
			if a.Val != ".l" {
				return false, ""
			}
		}
		if a.Key == "href" {
			return true, a.Val
		}
	}
	return false, ""
}

func parseAnswers(htmlReader io.Reader) (string, error)  {
	doc, err := goquery.NewDocumentFromReader(htmlReader)
	if err != nil {
		return "", err
	}
	firstAnswer := doc.Find(".answer").First()
	if firstAnswer != nil {
		preformat := firstAnswer.Find("pre").First()
		if preformat != nil {
			code := preformat.Find("code").First()
			if code != nil {
				return code.Text(), nil
			}
		}
		code := firstAnswer.Find("code").First()
		if code != nil {
			return code.Text(), nil
		}
		postText := firstAnswer.Find(".post-text").First()
		if postText != nil {
			return postText.Text(), nil
		}
	}
	return "", nil
}


