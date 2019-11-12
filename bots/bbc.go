package bots

import (
	"context"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/gocolly/colly"
	"github.com/gocolly/colly/debug"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// BBCExtract starts a bot for https://www.bbc.com/persian
func BBCExtract() {
	var data *NewsData = &NewsData{}
	collection := getDatabaseCollection("BBC")

	s := strings.Builder{}
	s.Grow(10000)
	linkExtractor := colly.NewCollector(
		colly.MaxDepth(3),
		colly.URLFilters(
			regexp.MustCompile(`https://www\.bbc\.com/persian`),
		),
		// colly.Async(true),
		colly.Debugger(&debug.LogDebugger{}),
	)
	newsRegex := regexp.MustCompile(`https://www\.bbc\.com/persian/\w+-\d{8}`)
	codeRegex := regexp.MustCompile(`\w+-\d{8}`)

	detailExtractor := linkExtractor.Clone()
	detailExtractor.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 1})

	linkExtractor.OnHTML("a[href]", func(e *colly.HTMLElement) {
		log.Println(e.Attr("href"))
		if newsRegex.MatchString(e.Request.AbsoluteURL(e.Attr("href"))) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			partialURL := e.Attr("href")
			filter := bson.M{"code": codeRegex.FindString(partialURL)}
			res := collection.FindOne(ctx, filter)

			code := struct {
				Code string
			}{}
			err := res.Decode(&code)
			if err != nil && err != mongo.ErrNoDocuments {
				log.Fatal(err)
			}
			if code.Code == "" {
				// go detailExtractor.Visit(e.Request.AbsoluteURL(e.Attr("href")))
				detailExtractor.Visit(e.Request.AbsoluteURL(e.Attr("href")))
			}
			// log.Println("Extractor is Skipping", e.Request.URL)
		}
		e.Request.Visit(e.Attr("href"))
	})

	detailExtractor.OnRequest(func(r *colly.Request) {
		data.Title = ""
		data.Summary = ""
		data.Text = ""
		data.Tags = make([]string, 0, 16)
		data.Code = ""
		data.DateTime = ""
		data.NewsAgency = ""
		data.Reporter = ""
	})

	// news title
	detailExtractor.OnHTML(".story-body__h1", func(e *colly.HTMLElement) {
		data.Title = strings.TrimSpace(e.Text)
	})

	// news summary
	detailExtractor.OnHTML(".story-body__introduction", func(e *colly.HTMLElement) {
		data.Summary = strings.TrimSpace(e.Text)
	})

	//news body
	detailExtractor.OnHTML(`p[class=""]`, func(e *colly.HTMLElement) {
		s.WriteString(strings.TrimSpace(e.Text))
	})

	// news code
	detailExtractor.OnResponse(func(r *colly.Response) {
		data.Code = codeRegex.FindString(r.Request.URL.String())
	})

	// news date and time
	detailExtractor.OnHTML("div[data-datetime]", func(e *colly.HTMLElement) {
		data.DateTime = e.Attr("data-datetime")
	})

	detailExtractor.OnScraped(func(r *colly.Response) {
		data.Text = s.String()
		data.NewsAgency = "خبرگزاری بی بی سی"

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		collection.InsertOne(ctx, data)
		log.Println("Scraped:", r.Request.URL.String())
	})
	linkExtractor.Visit("https://www.bbc.com/persian")
	// linkExtractor.Wait()
}
