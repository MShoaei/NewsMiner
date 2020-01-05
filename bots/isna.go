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
	isnaNewsRegex = regexp.MustCompile(`https://www\.isna\.ir/news/\d+/.*`)
	isnaCodeRegex = regexp.MustCompile(`\d{11}`)
)

// ISNAExtract starts a bot for https://www.isna.ir
func ISNAExtract(exportCmd chan<- *exec.Cmd) {
	collection := collectionInit("ISNA")

	var cmd = exec.Command("mongoexport",
		"--uri=mongodb://localhost:27017/ISNA",
		fmt.Sprintf("--collection=%s", collection.Name()),
		"--type=csv",
		"--fields=title,summary,text,tags,code,datetime,newsagency,reporter",
		fmt.Sprintf("--out=./isna/%s.csv", collection.Name()),
	)
	exportCmd <- cmd

	linkExtractor := colly.NewCollector(
		colly.MaxDepth(1),
		colly.URLFilters(
			//regexp.MustCompile(`https://www\.isna\.ir(|/news/\d+.*)$`),
			regexp.MustCompile(`https://www\.isna\.ir/page/archive\.xhtml.*`),
		),
		//colly.DisallowedDomains("leader.ir","imam-khomeini.ir", "president.ir", "parliran.ir"),
		//colly.Async(true),
		colly.Debugger(&debug.LogDebugger{}),
	)

	detailExtractor := colly.NewCollector()
	detailExtractor.MaxDepth = 1
	detailExtractor.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 5})

	linkExtractor.OnHTML(".items li a[href]", func(e *colly.HTMLElement) {
		log.Println(e.Attr("href"))
		if isnaNewsRegex.MatchString(e.Request.AbsoluteURL(e.Attr("href"))) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			partialURL := e.Attr("href")
			filter := bson.M{"code": isnaCodeRegex.FindString(partialURL)}
			res := collection.FindOne(ctx, filter)

			code := struct {
				Code string
			}{}
			err := res.Decode(&code)
			if err != nil && err != mongo.ErrNoDocuments {
				log.Fatal(err)
			}
			//code := checkNewsCode(e,codeRegex,collection)
			//if code == "" {
			if code.Code == "" {
				fmt.Println(e.Request.AbsoluteURL(e.Attr("href")))
				detailExtractor.Visit(e.Request.AbsoluteURL(e.Attr("href")))
			}
			// log.Println("Extractor is Skipping", e.Request.URL)
		}
		//e.Request.Visit(e.Attr("href"))
	})

	q, _ := queue.New(2, &queue.InMemoryQueueStorage{MaxSize: 700})
	for i := 1; i < 600; i++ {
		q.AddURL(fmt.Sprintf("https://www.isna.ir/page/archive.xhtml?mn=7&wide=0&dy=20&ms=0&pi=%d&yr=1397", i))
	}
	q.Run(linkExtractor)
}

func newISNANewsExtractor(data *NewsData, collection *mongo.Collection) *colly.Collector {
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
	detailExtractor.OnHTML(".first-title", func(e *colly.HTMLElement) {
		data.Title = strings.TrimSpace(e.Text)
	})

	// news summary
	detailExtractor.OnHTML(".summary", func(e *colly.HTMLElement) {
		data.Summary = strings.TrimSpace(e.Text)
	})

	//news body
	detailExtractor.OnHTML("div[itemprop=articleBody]", func(e *colly.HTMLElement) {
		data.Text = strings.TrimSpace(e.Text)
	})

	//news tags
	detailExtractor.OnHTML(".tags a", func(e *colly.HTMLElement) {
		data.Tags = append(data.Tags, strings.TrimSpace(e.Text))
	})

	// news code
	detailExtractor.OnResponse(func(r *colly.Response) {
		// fmt.Println(strings.Split(e.Text, ":")[1])
		// data.Code = strings.TrimSpace(strings.Split(e.Text, ":")[1])
		data.Code = isnaCodeRegex.FindString(r.Request.URL.String())
	})

	// news date and time
	detailExtractor.OnHTML(".fa-calendar-o~ .text-meta", func(e *colly.HTMLElement) {
		// fmt.Println(strings.Split(e.Text, ":")[1])
		// strings.Replace(e.Text, "/", "-", 1)
		data.DateTime = strings.TrimSpace(e.Text)
	})

	// reporter
	detailExtractor.OnHTML(".fa-edit~ strong", func(e *colly.HTMLElement) {
		data.Reporter = strings.TrimSpace(e.Text)
	})

	detailExtractor.OnScraped(func(r *colly.Response) {
		data.NewsAgency = "خبرگزاری ایسنا"

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		collection.InsertOne(ctx, data)
		log.Println("Scraped:", r.Request.URL.String())
	})
	return detailExtractor
}
