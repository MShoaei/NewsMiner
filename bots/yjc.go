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

// YJCExtract starts a bot for https://www.yjc.ir
func YJCExtract(exportCmd chan<- *exec.Cmd) {
	var data *NewsData = &NewsData{}

	//collection := getDatabaseCollection("YJC")
	collection := collectionInit("YJC")

	var cmd = exec.Command("mongoexport",
		"--uri=mongodb://localhost:27017/YJC",
		fmt.Sprintf("--collection=%s", collection.Name()),
		"--type=csv",
		"--fields=title,summary,text,tags,code,datetime,newsagency,reporter",
		fmt.Sprintf("--out=./yjc/%s.csv", collection.Name()),
	)
	exportCmd <- cmd

	linkExtractor := colly.NewCollector(
		colly.MaxDepth(1),
		colly.URLFilters(
			regexp.MustCompile(``),
		),
		// colly.Async(true),
		colly.Debugger(&debug.LogDebugger{}),
	)
	newsRegex := regexp.MustCompile(`https://www\.yjc\.ir/fa/news/\d+/.*`)
	codeRegex := regexp.MustCompile(`\d{7}`)

	detailExtractor := colly.NewCollector()
	detailExtractor.MaxDepth = 1
	detailExtractor.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 8})

	linkExtractor.OnHTML("a[href]", func(e *colly.HTMLElement) {
		//log.Println(e.Attr("href"))
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
		//e.Request.Visit(e.Attr("href"))
	})

	detailExtractor.OnRequest(func(r *colly.Request) {
		data.Title = ""
		data.Summary = ""
		data.Text = ""
		data.Tags = nil
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
	detailExtractor.OnHTML(".Htags_news_subtitle .news_strong", func(e *colly.HTMLElement) {
		data.Summary = strings.TrimSpace(e.Text)
	})

	//news body
	detailExtractor.OnHTML(".body", func(e *colly.HTMLElement) {
		data.Text = strings.TrimSpace(e.Text)
	})

	//news code
	detailExtractor.OnResponse(func(r *colly.Response) {
		data.Code = codeRegex.FindString(r.Request.URL.String())
	})

	// news date and time
	detailExtractor.OnHTML(".news_pdate_c", func(e *colly.HTMLElement) {
		// fmt.Println(strings.Split(e.Text, ":")[1])
		data.DateTime = strings.TrimSpace(strings.Split(e.Text, ":")[1])
	})

	detailExtractor.OnScraped(func(r *colly.Response) {
		data.NewsAgency = "باشگاه خبرنگاران جوان"

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		collection.InsertOne(ctx, data)
		log.Println("Scraped:", r.Request.URL.String())
	})

	q, _ := queue.New(2, &queue.InMemoryQueueStorage{MaxSize: 1300})

	for i := 1; i < 1200; i++ {
		q.AddURL(fmt.Sprintf("https://www.yjc.ir/fa/archive?service_id=-1&sec_id=-1&cat_id=-1&rpp=100&from_date=1390/01/01&to_date=1398/10/14&p=%d", i))
	}
	q.Run(linkExtractor)
}
