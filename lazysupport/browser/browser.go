package browser

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/chromedp"
	"github.com/microcosm-cc/bluemonday"

	_ "embed"
)

func getBrowser() (context.Context, context.CancelFunc) {
	allocCtx, _ := chromedp.NewExecAllocator(context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:],
			//chromedp.Flag("auto-open-devtools-for-tabs ", true),
			chromedp.Flag("ignore-certificate-errors", true),
			chromedp.WindowSize(1366, 768),
		)...,
	)

	log := bytes.NewBuffer([]byte{})

	printf := func(format string, vars ...any) {
		fmt.Fprintf(log, format+"\n", vars...)
	}
	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(printf))
	return ctx, cancel
}

func GetHTML(url string) (string, error) {
	ctx, cancel := getBrowser()
	defer cancel()

	var body string
	var err error
	tasks := chromedp.Tasks{
		chromedp.Navigate(url),
		chromedp.Sleep(2000 * time.Millisecond),
		chromedp.ActionFunc(func(ctx context.Context) error {
			node, err := dom.GetDocument().Do(ctx)
			if err != nil {
				return err
			}
			body, err = dom.GetOuterHTML().WithNodeID(node.NodeID).Do(ctx)
			return err
		}),
	}

	err = chromedp.Run(ctx, tasks)
	if err != nil {
		return "", err
	}

	return body, nil

}

//go:embed Readability.js
var readability string

//go:embed script.js
var script string

func GetArticle(url string) (*Article, error) {
	ctx, cancel := getBrowser()
	defer cancel()

	article := &Article{}
	var err error
	var res string

	tasks := chromedp.Tasks{
		chromedp.Navigate(url),
		chromedp.Sleep(2000 * time.Millisecond),
		chromedp.Evaluate(readability+script, &res),
	}
	err = chromedp.Run(ctx, tasks)
	if err != nil {
		return nil, err
	}
	if res == "" {
		time.Sleep(1500 * time.Millisecond)
		err = chromedp.Run(ctx, chromedp.Evaluate(readability+script, &res))
		if err != nil {
			panic(err)
		}
	}
	if res == "" {
		return nil, fmt.Errorf("could not get article")
	}

	err = json.Unmarshal([]byte(res), article)
	if err != nil {
		return nil, err
	}

	article.Content = bluemonday.UGCPolicy().Sanitize(article.Content)

	return article, nil
}

type Article struct {
	Title       string `json:"title"`
	ByLine      string `json:"byline"`
	Dir         string `json:"dir"`
	Lang        string `json:"lang"`
	Content     string `json:"content"`
	TextContent string `json:"textContent"`
	Length      uint   `json:"length"`
	Excerpt     string `json:"excerpt"`
	SiteName    string `json:"siteName"`
}
