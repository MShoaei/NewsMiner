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

// TabnakExtract starts a bot for https://www.tabnak.ir
func TabnakExtract() {
	var data *NewsData = &NewsData{}
	collection := getDatabaseCollection("Tabnak")

	linkExtractor := colly.NewCollector(
		colly.MaxDepth(3),
		colly.URLFilters(
			regexp.MustCompile(`https://www\.tabnak\.ir(|/fa/news/\d+.*)$`),
		),
		// colly.Async(true),
		colly.Debugger(&debug.LogDebugger{}),
	)
	newsRegex := regexp.MustCompile(`https://www\.tabnak\.ir/fa/news/\d+/.*`)
	codeRegex := regexp.MustCompile(`\d{6}`)

	detailExtractor := linkExtractor.Clone()
	detailExtractor.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 1})

	linkExtractor.OnHTML("a[href]", func(e *colly.HTMLElement) {
		// log.Println(e.Attr("href"))
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
	detailExtractor.OnHTML(".title", func(e *colly.HTMLElement) {
		data.Title = strings.TrimSpace(e.Text)
	})

	// news summary
	detailExtractor.OnHTML(".subtitle", func(e *colly.HTMLElement) {
		data.Summary = strings.TrimSpace(e.Text)
	})

	//news body
	detailExtractor.OnHTML(".body", func(e *colly.HTMLElement) {
		data.Text = strings.TrimSpace(e.Text)
	})

	//news tags
	detailExtractor.OnHTML(".btn-primary-news", func(e *colly.HTMLElement) {
		data.Tags = append(data.Tags, strings.TrimSpace(e.Text))
	})

	// news code
	detailExtractor.OnResponse(func(r *colly.Response) {
		// fmt.Println(strings.Split(e.Text, ":")[1])
		// data.Code = strings.TrimSpace(strings.Split(e.Text, ":")[1])
		data.Code = codeRegex.FindString(r.Request.URL.String())
	})

	// news date and time
	detailExtractor.OnHTML(".fa_date", func(e *colly.HTMLElement) {
		// fmt.Println(strings.Split(e.Text, ":")[1])
		data.DateTime = strings.TrimSpace(e.Text)
	})

	detailExtractor.OnScraped(func(r *colly.Response) {
		data.NewsAgency = "خبرگزاری تابناک"

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		collection.InsertOne(ctx, data)
		log.Println("Scraped:", r.Request.URL.String())
	})
	linkExtractor.Visit("https://www.tabnak.ir")
	// linkExtractor.Wait()
}
