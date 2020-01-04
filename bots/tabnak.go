package bots

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/gocolly/colly"
	"github.com/gocolly/colly/debug"
	"github.com/gocolly/colly/queue"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// TabnakExtract starts a bot for https://www.tabnak.ir
func TabnakExtract(exportCmd chan<- *exec.Cmd) {
	var data *NewsData

	//collection := getDatabaseCollection("Tabnak")
	collection := collectionInit("Tabnak")

	var cmd = exec.Command("mongoexport",
		"--uri=mongodb://localhost:27017/Tabnak",
		fmt.Sprintf("--collection=%s", collection.Name()),
		"--type=csv",
		"--fields=title,summary,text,tags,code,datetime,newsagency,reporter",
		fmt.Sprintf("--out=./tabnak/%s.csv", collection.Name()),
	)
	exportCmd <- cmd

	linkExtractor := colly.NewCollector(
		colly.MaxDepth(1),
		colly.URLFilters(
			//regexp.MustCompile(`https://www\.tabnak\.ir(|/fa/news/\d+.*)$`),
			//regexp.MustCompile(`http(|s)://www\.tabnak\.ir`),
			//regexp.MustCompile(`http://ostanha\.tabnak\.ir`),
			//regexp.MustCompile(`http://(|www\.)tabnak\w+\.ir`),
			regexp.MustCompile(`https://www.tabnak.ir/fa/archive.*`),
		),
		//colly.Async(true),
		colly.Debugger(&debug.LogDebugger{}),
	)

	newsRegex := regexp.MustCompile(`http(|s)://(www|ostanha)\.tabnak\w*\.ir/fa/news/\d+/.*`)
	codeRegex := regexp.MustCompile(`\d{6}`)

	detailExtractor := colly.NewCollector()
	detailExtractor.MaxDepth = 1
	detailExtractor.Async = true
	detailExtractor.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 8})

	linkExtractor.OnHTML(".linear_news a[href]", func(e *colly.HTMLElement) {
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
				fmt.Println(e.Request.AbsoluteURL(e.Attr("href")))
				detailExtractor.Visit(e.Request.AbsoluteURL(e.Attr("href")))
			}
			// log.Println("Extractor is Skipping", e.Request.URL)
		}
		//e.Request.Visit(e.Attr("href"))
	})

	detailExtractor.OnRequest(func(r *colly.Request) {
		data = &NewsData{}
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

	q, _ := queue.New(3, &queue.InMemoryQueueStorage{MaxSize: 1300})
	for i := 1; i < 1200; i++ {
		q.AddURL(fmt.Sprintf("https://www.tabnak.ir/fa/archive?service_id=-1&sec_id=-1&cat_id=-1&rpp=100&from_date=1384/01/01&to_date=1398/10/13&p=%d", i))
	}
	q.Run(linkExtractor)
}
