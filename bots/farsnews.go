package bots

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/debug"
	"github.com/gocolly/colly/v2/queue"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	farsnewsNewsRegex = regexp.MustCompile(`https://www\.farsnews\.ir/.*news/\d+/.*`)
	farsnewsCodeRegex = regexp.MustCompile(`\d+`)
)

// FarsNewsExtract starts a bot for https://www.farsnews.ir
func FarsNewsExtract(exportCmd chan<- *exec.Cmd) {
	collection := collectionInit("Farsnews")

	var cmd = exec.Command("mongoexport",
		"--uri=mongodb://localhost:27017/Farsnews",
		fmt.Sprintf("--collection=%s", collection.Name()),
		"--type=csv",
		"--fields=title,summary,text,tags,code,datetime,newsagency,reporter",
		fmt.Sprintf("--out=./farsnews/%s.csv", collection.Name()),
	)
	exportCmd <- cmd

	linkExtractor := colly.NewCollector(
		colly.MaxDepth(1),
		colly.URLFilters(
		//regexp.MustCompile(`https://www\.farsnews\.ir(|/news/\d+.*)$`),
		),
		// colly.Async(true),
		colly.Debugger(&debug.LogDebugger{}),
	)

	linkExtractor.OnHTML("li.media>a[href]", func(e *colly.HTMLElement) {
		// log.Println(e.Attr("href"))
		if farsnewsNewsRegex.MatchString(e.Request.AbsoluteURL(e.Attr("href"))) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			partialURL := e.Attr("href")
			filter := bson.M{"code": farsnewsCodeRegex.FindString(partialURL)}
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
				extractor := newFarsnewsDetailExtractor(d, collection)
				extractor.Visit(e.Request.AbsoluteURL(e.Attr("href")))
			}
			// log.Println("Extractor is Skipping", e.Request.URL)
		}
		// e.Request.Visit(e.Attr("href"))
	})

	q, _ := queue.New(6, &queue.InMemoryQueueStorage{MaxSize: 1200})
	for month := 10; month > 0; month-- {
		for day := 15; day > 0; day-- {
			for i := 1; i < 31; i++ {
				q.AddURL(fmt.Sprintf("https://www.farsnews.ir/archive?cat=-1&subcat=-1&date=1398%%2F%d%%2F%d&p=%d", month, day, i))
			}
		}
	}
	q.Run(linkExtractor)
}

func newFarsnewsDetailExtractor(data *NewsData,
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
		data.Code = farsnewsCodeRegex.FindString(r.Request.URL.String())
	})

	// news date and time
	detailExtractor.OnHTML(".publish-time span", func(e *colly.HTMLElement) {
		data.DateTime = e.Text
	})

	detailExtractor.OnScraped(func(r *colly.Response) {
		data.NewsAgency = "خبرگزاری فارس"
		data.DateTime = strings.ReplaceAll(data.DateTime, "\u00a0", " ")

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
