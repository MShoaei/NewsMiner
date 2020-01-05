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

var (
	tabnakNewsRegex = regexp.MustCompile(`http(|s)://(www|ostanha)\.tabnak\w*\.ir/fa/news/\d+/.*`)
	tabnakCodeRegex = regexp.MustCompile(`\d{6}`)
)

// TabnakExtract starts a bot for https://www.tabnak.ir
func TabnakExtract(exportCmd chan<- *exec.Cmd) {
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
			regexp.MustCompile(`https://www.tabnak.ir/fa/archive.*`),
		),
		//colly.Async(true),
		colly.Debugger(&debug.LogDebugger{}),
	)

	linkExtractor.OnHTML(".linear_news a[href]", func(e *colly.HTMLElement) {
		log.Println(e.Attr("href"))
		if tabnakNewsRegex.MatchString(e.Request.AbsoluteURL(e.Attr("href"))) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			partialURL := e.Attr("href")
			filter := bson.M{"code": tabnakCodeRegex.FindString(partialURL)}
			res := collection.FindOne(ctx, filter)

			code := struct {
				Code string
			}{}
			err := res.Decode(&code)
			if err != nil && err != mongo.ErrNoDocuments {
				log.Fatal(err)
			}
			if code.Code == "" {
				d := &NewsData{}
				extractor := newTabnakDetailExtractor(d, collection)
				extractor.Visit(e.Request.AbsoluteURL(e.Attr("href")))
			}
			// log.Println("Extractor is Skipping", e.Request.URL)
		}
		//e.Request.Visit(e.Attr("href"))
	})

	q, _ := queue.New(6, &queue.InMemoryQueueStorage{MaxSize: 1300})
	for i := 1; i < 1200; i++ {
		q.AddURL(fmt.Sprintf("https://www.tabnak.ir/fa/archive?service_id=-1&sec_id=-1&cat_id=-1&rpp=100&from_date=1384/01/01&to_date=1398/10/13&p=%d", i))
	}
	q.Run(linkExtractor)
}

func newTabnakDetailExtractor(data *NewsData,
	collection *mongo.Collection) *colly.Collector {
	detailExtractor := colly.NewCollector()
	detailExtractor.MaxDepth = 1
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
		data.Code = tabnakCodeRegex.FindString(r.Request.URL.String())
	})

	// news date and time
	detailExtractor.OnHTML(".fa_date", func(e *colly.HTMLElement) {
		data.DateTime = strings.TrimSpace(e.Text)
	})

	detailExtractor.OnScraped(func(r *colly.Response) {
		data.NewsAgency = "خبرگزاری تابناک"

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		_, err := collection.InsertOne(ctx, data)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Scraped:", r.Request.URL.String())
	})
	return detailExtractor
}
