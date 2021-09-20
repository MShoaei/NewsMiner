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

var ()

type Tabnak struct {
	bot
}

func NewTabnakBot(threads int, db *database.DB, collection *mongo.Collection, wg *sync.WaitGroup) *Tabnak {
	tabnakNewsRegex, _ := db.GetNewsPageRegex(string(TabnakAgency))
	tabnakCodeRegex, _ := db.GetNewsCodeRegex(string(TabnakAgency))
	archiveURL, _ := db.GetArchiveURL(string(TabnakAgency))
	t := &Tabnak{
		bot: bot{
			wg:         wg,
			NewsPage:   tabnakNewsRegex,
			NewsCode:   tabnakCodeRegex,
			DB:         db,
			Collection: collection,
			Threads:    threads,
			ArchiveURL: archiveURL,
		},
	}
	t.ArchiveCrawler = t.getArchiveCrawler()
	return t
}

func (t *Tabnak) Extract(pages int) {
	defer t.wg.Done()
	q := t.fillQueue(pages)
	q.Run(t.ArchiveCrawler)
}

func (t *Tabnak) fillQueue(pages int) *queue.Queue {
	now := ptime.New(time.Now())
	q, _ := queue.New(t.Threads, &queue.InMemoryQueueStorage{MaxSize: pages})

	currentPage := 0
	for currentPage < pages {
		count := t.getPageCount(now)
		for i := 1; i <= count; i++ {
			err := q.AddURL(fmt.Sprintf(t.ArchiveURL, now.Format("yyyy/MM/dd"), i))
			if err != nil {
				return q
			}
			currentPage++
		}
		now = now.AddDate(0, 0, -1)
	}
	return q
}

func (t *Tabnak) getPageCount(date ptime.Time) int {
	return 10
}

func (t *Tabnak) isNewsPage(e *colly.HTMLElement) bool {
	return t.NewsPage.MatchString(e.Request.AbsoluteURL(e.Attr("href")))
}

// TabnakExtract starts a bot for https://www.tabnak.ir
func (t *Tabnak) getArchiveCrawler() *colly.Collector {
	linkExtractor := colly.NewCollector(
		colly.MaxDepth(1),
	)

	linkExtractor.OnHTML(".linear_news a[href]", func(e *colly.HTMLElement) {
		partialURL := e.Attr("href")
		code := t.NewsCode.FindString(partialURL)
		if !t.isNewsPage(e) || t.DB.NewsWithCodeExists(t.Collection, code) {
			return
		}
		data, err := t.extractPageDetail(e.Request.AbsoluteURL(e.Attr("href")))
		if err != nil {
			return
		}
		t.DB.Save(t.Collection, data)
	})

	return linkExtractor
}

func (t *Tabnak) extractPageDetail(url string) (*NewsData, error) {
	pageCrawler := colly.NewCollector(
		colly.MaxDepth(1),
	)

	data := &NewsData{
		NewsAgencyID: "Tabnak",
		Tags:         make([]string, 0, 8),
	}

	// news title
	pageCrawler.OnHTML(".title", func(e *colly.HTMLElement) {
		data.Title = strings.TrimSpace(e.Text)
	})

	// news summary
	pageCrawler.OnHTML(".subtitle", func(e *colly.HTMLElement) {
		data.Summary = strings.TrimSpace(e.Text)
	})

	//news body
	pageCrawler.OnHTML(".body", func(e *colly.HTMLElement) {
		data.Text = strings.TrimSpace(e.Text)
	})

	//news tags
	pageCrawler.OnHTML(".btn-primary-news", func(e *colly.HTMLElement) {
		data.Tags = append(data.Tags, strings.TrimSpace(e.Text))
	})

	// news code
	pageCrawler.OnResponse(func(r *colly.Response) {
		data.Code = t.NewsCode.FindString(r.Request.URL.String())
	})

	// news date and time
	pageCrawler.OnHTML(".fa_date", func(e *colly.HTMLElement) {
		data.DateTime = strings.TrimSpace(e.Text)
	})

	err := pageCrawler.Visit(url)
	return data, err
}
