package bots

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/gocolly/colly"
	"github.com/gocolly/colly/debug"
	"github.com/zolamk/colly-mongo-storage/colly/mongo"
)

// YJCExtract starts a bot for https://www.yjc.ir
func YJCExtract() {
	var data *NewsData = &NewsData{}
	linkExtractor := colly.NewCollector(
		colly.MaxDepth(2),
		colly.URLFilters(
			regexp.MustCompile(`https://www\.yjc\.ir(|/fa/news/\d+.*)$`),
		),
		colly.Debugger(&debug.LogDebugger{}),
	)
	newsRegex := regexp.MustCompile(`https://www\.yjc\.ir/fa/news/\d+/.*`)
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
		// log.Println("Visiting ", r.URL)
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
		data.Tags = []string{}
		data.Code = ""
		data.DateTime = ""
		data.NewsAgency = ""
		data.Reporter = ""
	})

	// news code
	detailExtractor.OnHTML(".news_id_c", func(e *colly.HTMLElement) {
		// fmt.Println(strings.Split(e.Text, ":")[1])
		data.Code = strings.TrimSpace(strings.Split(e.Text, ":")[1])
	})

	// news date and time
	detailExtractor.OnHTML(".news_pdate_c", func(e *colly.HTMLElement) {
		// fmt.Println(strings.Split(e.Text, ":")[1])
		data.DateTime = strings.TrimSpace(strings.Split(e.Text, ":")[1])
	})

	// news title
	detailExtractor.OnHTML(".title", func(e *colly.HTMLElement) {
		data.Title = strings.TrimSpace(e.Text)
	})

	// news summary
	detailExtractor.OnHTML(".Htags_news_subtitle .news_strong", func(e *colly.HTMLElement) {
		data.Summary = strings.TrimSpace(e.Text)
	})

	//news body
	detailExtractor.OnHTML(".body", func(e *colly.HTMLElement) {
		data.Text = strings.TrimSpace(e.Text)
	})
	detailExtractor.OnScraped(func(_ *colly.Response) {
		data.NewsAgency = "باشگاه خبرنگاران جوان"
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		collection.InsertOne(ctx, data)
	})
	linkExtractor.Visit("https://www.yjc.ir")
}
