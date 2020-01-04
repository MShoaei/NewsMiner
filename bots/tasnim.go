package bots

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/gocolly/colly/queue"

	"github.com/gocolly/colly/debug"

	"github.com/gocolly/colly"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// TasnimExtract starts a bot for https://www.tasnimnews.com
func TasnimExtract(exportCmd chan<- *exec.Cmd) {
	var data *NewsData

	//collection := getDatabaseCollection("Tasnim")
	collection := collectionInit("Tasnim")

	var cmd = exec.Command("mongoexport",
		"--uri=mongodb://localhost:27017/Tasnim",
		fmt.Sprintf("--collection=%s", collection.Name()),
		"--type=csv",
		"--fields=title,summary,text,tags,code,datetime,newsagency,reporter",
		fmt.Sprintf("--out=./tasnim/%s.csv", collection.Name()),
	)
	exportCmd <- cmd

	linkExtractor := colly.NewCollector(
		colly.MaxDepth(1),
		colly.URLFilters(
			// regexp.MustCompile(`https://www\.tasnimnews\.com(|/fa/news/\d+.*)$`),
			regexp.MustCompile(`https://www.tasnimnews.com/fa/archive.*`),
		),
		// colly.Async(true),
		colly.Debugger(&debug.LogDebugger{}),
	)

	newsRegex := regexp.MustCompile(`https://www\.tasnimnews\.com/fa/news/\d+/.*`)
	codeRegex := regexp.MustCompile(`\d{7}`)

	detailExtractor := colly.NewCollector()
	detailExtractor.MaxDepth = 1
	detailExtractor.Async = true
	detailExtractor.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 8})

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
		//e.Request.Visit(e.Attr("href"))
	})

	detailExtractor.OnRequest(func(r *colly.Request) {
		data = &NewsData{}
	})

	// news title
	detailExtractor.OnHTML("h1", func(e *colly.HTMLElement) {
		data.Title = strings.TrimSpace(e.Text)
	})

	// news summary
	detailExtractor.OnHTML(".lead", func(e *colly.HTMLElement) {
		data.Summary = strings.TrimSpace(e.Text)
	})

	//news body
	detailExtractor.OnHTML(".story", func(e *colly.HTMLElement) {
		data.Text = strings.TrimSpace(e.Text)
	})

	//news tags
	// detailExtractor.OnHTML(".btn-primary-news", func(e *colly.HTMLElement) {
	// 	data.Tags = append(data.Tags, strings.TrimSpace(e.Text))
	// })

	// news code
	detailExtractor.OnResponse(func(r *colly.Response) {
		// fmt.Println(strings.Split(e.Text, ":")[1])
		// data.Code = strings.TrimSpace(strings.Split(e.Text, ":")[1])
		data.Code = codeRegex.FindString(r.Request.URL.String())
	})

	// news date and time
	detailExtractor.OnHTML(".time", func(e *colly.HTMLElement) {
		// fmt.Println(strings.Split(e.Text, ":")[1])
		data.DateTime = strings.TrimSpace(e.Text)
	})

	detailExtractor.OnScraped(func(r *colly.Response) {
		data.NewsAgency = "خبرگزاری تسنیم"

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		collection.InsertOne(ctx, data)
		log.Println("Scraped:", r.Request.URL.String())
	})

	q, _ := queue.New(2, &queue.InMemoryQueueStorage{MaxSize: 10000})

	for month := 5; month > 0; month-- {
		for day := 30; day > 0; day-- {
			for page := 1; page < 40; page++ {
				q.AddURL(fmt.Sprintf("https://www.tasnimnews.com/fa/archive?date=1398%%2F%d%%2F%d&sub=-1&service=-1&page=%d", month, day, page))
			}
		}
	}
	q.Run(linkExtractor)
}
