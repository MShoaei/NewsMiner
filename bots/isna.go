package bots

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/MShoaei/NewsMiner/database"
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/queue"
	ptime "github.com/yaa110/go-persian-calendar"
	"go.mongodb.org/mongo-driver/mongo"
)

type ISNA struct {
	bot
}

func NewISNABot(threads int, db *database.DB, collection *mongo.Collection, wg *sync.WaitGroup) *ISNA {
	isnaNewsRegex, _ := db.GetNewsPageRegex(string(ISNAAgency))
	isnaCodeRegex, _ := db.GetNewsCodeRegex(string(ISNAAgency))
	archiveURL, _ := db.GetArchiveURL(string(ISNAAgency))
	i := &ISNA{
		bot: bot{
			wg:         wg,
			NewsPage:   isnaNewsRegex,
			NewsCode:   isnaCodeRegex,
			DB:         db,
			Collection: collection,
			Threads:    threads,
			ArchiveURL: archiveURL,
		},
	}
	i.ArchiveCrawler = i.getArchiveCrawler()
	return i
}

// ISNAExtract starts a bot for https://www.isna.ir
func (i *ISNA) Extract(pages int) {
	defer i.wg.Done()
	q := i.fillQueue(pages)
	q.Run(i.ArchiveCrawler)
}

func (i *ISNA) fillQueue(pages int) *queue.Queue {
	now := ptime.New(time.Now())
	q, _ := queue.New(i.Threads, &queue.InMemoryQueueStorage{MaxSize: pages})

	currentPage := 0
	for currentPage < pages {
		count := i.getPageCount(now)
		for index := 0; index < count; index++ {
			err := q.AddURL(fmt.Sprintf(i.ArchiveURL, now.Month(), now.Day(), index, now.Year()))
			if err != nil {
				return q
			}
			currentPage++
		}
		now = now.AddDate(0, 0, -1)
	}
	return q
}

func (i *ISNA) getPageCount(date ptime.Time) int {
	c := colly.NewCollector(
		colly.MaxDepth(1),
	)
	count := -2
	c.OnHTML(".pagination > li", func(e *colly.HTMLElement) {
		count++
	})
	c.Visit(fmt.Sprintf(i.ArchiveURL, date.Month(), date.Day(), 1, date.Year()))
	return count
}

func (i *ISNA) isNewsPage(e *colly.HTMLElement) bool {
	return i.NewsPage.MatchString(e.Request.AbsoluteURL(e.Attr("href")))
}

func (i *ISNA) getArchiveCrawler() *colly.Collector {
	linkExtractor := colly.NewCollector(
		colly.MaxDepth(1),
	)

	linkExtractor.OnHTML(".items li a[href]", func(e *colly.HTMLElement) {
		partialURL := e.Attr("href")
		code := i.NewsCode.FindString(partialURL)
		if !i.isNewsPage(e) || i.DB.NewsWithCodeExists(i.Collection, code) {
			return
		}

		data, err := i.extractPageDetail(e.Request.AbsoluteURL(e.Attr("href")))
		if err != nil {
			return
		}
		i.DB.Save(i.Collection, data)
	})
	return linkExtractor
}

func (i *ISNA) extractPageDetail(url string) (*NewsData, error) {
	pageCrawler := colly.NewCollector(
		colly.MaxDepth(1),
	)

	data := &NewsData{
		NewsAgencyID: "ISNA",
		Tags:         make([]string, 0, 8),
	}

	// news title
	pageCrawler.OnHTML(".first-title", func(e *colly.HTMLElement) {
		data.Title = strings.TrimSpace(e.Text)
	})

	// news summary
	pageCrawler.OnHTML(".summary", func(e *colly.HTMLElement) {
		data.Summary = strings.TrimSpace(e.Text)
	})

	//news body
	pageCrawler.OnHTML("div[itemprop=articleBody]", func(e *colly.HTMLElement) {
		data.Text = strings.TrimSpace(e.Text)
	})

	//news tags
	pageCrawler.OnHTML(".tags a", func(e *colly.HTMLElement) {
		data.Tags = append(data.Tags, strings.TrimSpace(e.Text))
	})

	// news code
	pageCrawler.OnResponse(func(r *colly.Response) {
		data.Code = i.NewsCode.FindString(r.Request.URL.String())
	})

	// news date and time
	pageCrawler.OnHTML(".fa-calendar-o~ .text-meta", func(e *colly.HTMLElement) {
		data.DateTime = strings.TrimSpace(e.Text)
	})

	// reporter
	pageCrawler.OnHTML(".fa-edit~ strong", func(e *colly.HTMLElement) {
		data.Reporter = strings.TrimSpace(e.Text)
	})

	err := pageCrawler.Visit(url)
	return data, err
}
