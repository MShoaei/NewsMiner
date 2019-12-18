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
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// FarsNewsExtract starts a bot for https://www.farsnews.com
func FarsNewsExtract(exportCmd chan<- *exec.Cmd) {
	var data *NewsData = &NewsData{}
	collection := getDatabaseCollection("Farsnews")

	var cmd = exec.Command("mongoexport",
		"--uri=mongodb://localhost:27017/Farsnews",
		fmt.Sprintf("--collection=%s", collection.Name()),
		"--type=csv",
		"--fields=title,summary,text,tags,code,datetime,newsagency,reporter",
		fmt.Sprintf("--out=./farsnews/farsnews%s.csv", collection.Name()),
	)
	exportCmd <- cmd

	linkExtractor := colly.NewCollector(
		colly.MaxDepth(3),
		colly.URLFilters(
			regexp.MustCompile(`https://www\.farsnews\.com(|/news/\d+.*)$`),
		),
		// colly.Async(true),
		// colly.Debugger(&debug.LogDebugger{}),
	)
	newsRegex := regexp.MustCompile(`https://www\.farsnews\.com/news/\d+/.*`)
	codeRegex := regexp.MustCompile(`\d{14}`)

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

	detailExtractor.OnScraped(func(r *colly.Response) {
		var err error
		data.NewsAgency = "خبرگزاری فارس"
		data.DateTime = strings.ReplaceAll(data.DateTime, "\u00a0", " ")

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		_, err = collection.InsertOne(ctx, data)
		if err != nil {
			log.Println(err)
		}
		log.Println("Scraped:", r.Request.URL.String())
	})
	linkExtractor.Visit("https://www.farsnews.com")
	// linkExtractor.Wait()
}
