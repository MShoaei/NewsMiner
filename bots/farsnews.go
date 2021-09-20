package bots

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MShoaei/NewsMiner/database"
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/queue"
	ptime "github.com/yaa110/go-persian-calendar"
	"go.mongodb.org/mongo-driver/mongo"
)

type Farsnews struct {
	bot
}

func NewFarsnewsBot(threads int, db *database.DB, collection *mongo.Collection, wg *sync.WaitGroup) *Farsnews {
	farsnewsNewsRegex, _ := db.GetNewsPageRegex(string(FarsnewsAgency))
	farsnewsCodeRegex, _ := db.GetNewsCodeRegex(string(FarsnewsAgency))
	archiveURL, _ := db.GetArchiveURL(string(FarsnewsAgency))
	f := &Farsnews{
		bot: bot{
			wg:         wg,
			NewsPage:   farsnewsNewsRegex,
			NewsCode:   farsnewsCodeRegex,
			DB:         db,
			Collection: collection,
			Threads:    threads,
			ArchiveURL: archiveURL,
		},
	}
	f.ArchiveCrawler = f.getArchiveCrawler()
	return f
}

func (f *Farsnews) Extract(pages int) {
	defer f.wg.Done()
	q := f.fillQueue(pages)
	q.Run(f.ArchiveCrawler)
}

// FarsNewsExtract starts a bot for https://www.farsnews.ir
func (f *Farsnews) fillQueue(pages int) *queue.Queue {
	now := ptime.New(time.Now())
	q, _ := queue.New(f.Threads, &queue.InMemoryQueueStorage{MaxSize: pages})

	currentPage := 0
	for currentPage < pages {
		count := f.getPageCount(now)
		for i := 1; i <= count; i++ {
			err := q.AddURL(fmt.Sprintf(f.ArchiveURL, now.Format("yyyy/MM/dd"), i))
			if err != nil {
				return q
			}
			currentPage++
		}
		now = now.AddDate(0, 0, -1)
	}
	return q
}

func (f *Farsnews) getPageCount(date ptime.Time) int {
	c := colly.NewCollector(
		colly.MaxDepth(1),
	)
	count := 0
	c.OnHTML("#NewsCount", func(e *colly.HTMLElement) {
		count, _ = strconv.Atoi(e.Attr("value"))
	})
	c.Visit(fmt.Sprintf(f.ArchiveURL, date.Format("yyyy/MM/dd"), 1))
	return count
}

func (f *Farsnews) isNewsPage(e *colly.HTMLElement) bool {
	return f.NewsPage.MatchString(e.Request.AbsoluteURL(e.Attr("href")))
}

func (f *Farsnews) getArchiveCrawler() *colly.Collector {
	linkExtractor := colly.NewCollector(
		colly.MaxDepth(1),
	)

	linkExtractor.OnHTML("li.media>a[href]", func(e *colly.HTMLElement) {
		// log.Println(e.Attr("href"))
		partialURL := e.Attr("href")
		code := f.NewsCode.FindString(partialURL)
		if !f.isNewsPage(e) || f.DB.NewsWithCodeExists(f.Collection, code) {
			return
		}

		data, err := f.extractPageDetail(e.Request.AbsoluteURL(e.Attr("href")))
		if err != nil {
			return
		}
		f.DB.Save(f.Collection, data)
	})
	return linkExtractor
}

func (f *Farsnews) extractPageDetail(url string) (*NewsData, error) {
	pageCrawler := colly.NewCollector(
		colly.MaxDepth(1),
	)

	data := &NewsData{
		NewsAgencyID: "Farsnews",
		Tags:         make([]string, 0, 8),
	}

	// news title
	pageCrawler.OnHTML(".d-block.text-justify", func(e *colly.HTMLElement) {
		data.Title = strings.TrimSpace(e.Text)
	})

	// news summary
	pageCrawler.OnHTML(".p-2", func(e *colly.HTMLElement) {
		data.Summary = strings.TrimSpace(e.Text)
	})

	//news body
	pageCrawler.OnHTML(".nt-body.text-right", func(e *colly.HTMLElement) {
		data.Text = strings.TrimSpace(e.Text)
	})

	//news tags
	pageCrawler.OnHTML(".ml-2.mb-2", func(e *colly.HTMLElement) {
		data.Tags = append(data.Tags, strings.TrimSpace(e.Text))
	})

	// news code
	pageCrawler.OnResponse(func(r *colly.Response) {
		data.Code = f.NewsCode.FindString(r.Request.URL.String())
	})

	// news date and time
	pageCrawler.OnHTML(".publish-time > time", func(e *colly.HTMLElement) {
		data.DateTime = strings.TrimSpace(e.Attr("datetime"))
	})

	err := pageCrawler.Visit(url)
	return data, err
}
