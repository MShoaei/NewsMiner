package bots

import (
	"context"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/gocolly/colly"
	"github.com/gocolly/colly/debug"
	"github.com/zolamk/colly-mongo-storage/colly/mongo"
)

// FarsNewsExtract starts a bot for https://www.farsnews.com
func FarsNewsExtract() {
	// var tags []string = make([]string, 16)
	var data *NewsData = &NewsData{}
	linkExtractor := colly.NewCollector(
		colly.MaxDepth(2),
		colly.URLFilters(
			regexp.MustCompile(`https://www\.farsnews\.com(|/news/\d+.*)$`),
		),
		colly.Debugger(&debug.LogDebugger{}),
	)
	newsRegex := regexp.MustCompile(`https://www\.farsnews\.com/news/\d+/.*`)
	codeRegex := regexp.MustCompile(`\d{14}`)
	storage := &mongo.Storage{
		Database: "colly",
		URI:      "mongodb://miner:password@localhost:27017/colly?authSource=admin&compressors=disabled&gssapiServiceName=mongodb",
	}
	if err := linkExtractor.SetStorage(storage); err != nil {
		panic(err)
	}

	detailExtractor := linkExtractor.Clone()
	detailExtractor.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 1})

	linkExtractor.OnRequest(func(r *colly.Request) {
		log.Println("Visiting ", r.URL)
	})

	linkExtractor.OnHTML("a[href]", func(e *colly.HTMLElement) {
		if newsRegex.MatchString(e.Request.AbsoluteURL(e.Attr("href"))) {
			// go detailExtractor.Visit(e.Request.AbsoluteURL(e.Attr("href")))
			detailExtractor.Visit(e.Request.AbsoluteURL(e.Attr("href")))
		}
		e.Request.Visit(e.Attr("href"))
	})

	detailExtractor.OnRequest(func(r *colly.Request) {
		data.Title = ""
		data.Summary = ""
		data.Text = ""
		data.Tags = make([]string, 0, 8)
		data.Code = ""
		data.DateTime = ""
		data.NewsAgency = ""
		data.Reporter = ""
	})

	// news title
	detailExtractor.OnHTML(".d-block.text-justify", func(e *colly.HTMLElement) {
		data.Title = strings.TrimSpace(e.Text)
	})

	// news summary
	detailExtractor.OnHTML(".p-2", func(e *colly.HTMLElement) {
		data.Summary = strings.TrimSpace(e.Text)
	})

	//news body
	detailExtractor.OnHTML(".nt-body.text-right", func(e *colly.HTMLElement) {
		data.Text = strings.TrimSpace(e.Text)
	})

	//news tags
	detailExtractor.OnHTML(".ml-2.mb-2", func(e *colly.HTMLElement) {
		data.Tags = append(data.Tags, strings.TrimSpace(e.Text))
	})

	// news code
	detailExtractor.OnResponse(func(r *colly.Response) {
		// fmt.Println(strings.Split(e.Text, ":")[1])
		// data.Code = strings.TrimSpace(strings.Split(e.Text, ":")[1])
		data.Code = codeRegex.FindString(r.Request.URL.String())
	})

	// news date and time
	detailExtractor.OnHTML(".publish-time span", func(e *colly.HTMLElement) {
		// fmt.Println(strings.Split(e.Text, ":")[1])
		data.DateTime += e.Text
	})

	detailExtractor.OnScraped(func(_ *colly.Response) {
		data.NewsAgency = "خبرگزاری فارس"
		data.DateTime = strings.ReplaceAll(data.DateTime, "\u00a0", " ")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		collection.InsertOne(ctx, data)
	})
	linkExtractor.Visit("https://www.farsnews.com")
}
