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

type Tasnim struct {
	bot
}

func NewTasnimBot(threads int, db *database.DB, collection *mongo.Collection, wg *sync.WaitGroup) *Tasnim {
	tasnimNewsRegex, _ := db.GetNewsPageRegex(string(TasnimAgency))
	tasnimCodeRegex, _ := db.GetNewsCodeRegex(string(TasnimAgency))
	archiveURL, _ := db.GetArchiveURL(string(TasnimAgency))

	t := &Tasnim{
		bot: bot{
			wg:         wg,
			NewsPage:   tasnimNewsRegex,
			NewsCode:   tasnimCodeRegex,
			DB:         db,
			Collection: collection,
			Threads:    threads,
			ArchiveURL: archiveURL,
		},
	}
	t.ArchiveCrawler = t.getArchiveCrawler()
	return t
}
func (t *Tasnim) Extract(pages int) {
	defer t.wg.Done()
	q := t.fillQueue(pages)
	q.Run(t.ArchiveCrawler)
}

func (t *Tasnim) fillQueue(pages int) *queue.Queue {
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

func (t *Tasnim) getPageCount(date ptime.Time) int {
	return 10
}

func (t *Tasnim) isNewsPage(e *colly.HTMLElement) bool {
	return t.NewsPage.MatchString(e.Request.AbsoluteURL(e.Attr("href")))
}

// TasnimExtract starts a bot for https://www.tasnimnews.com
func (t *Tasnim) getArchiveCrawler() *colly.Collector {
	linkExtractor := colly.NewCollector(
		colly.MaxDepth(1),
	)

	linkExtractor.OnHTML("article.list-item > a", func(e *colly.HTMLElement) {
		// log.Println(e.Attr("href"))
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

func (t *Tasnim) extractPageDetail(url string) (*NewsData, error) {
	pageCrawler := colly.NewCollector(
		colly.MaxDepth(1),
	)

	data := &NewsData{
		NewsAgencyID: "Tasnim",
		Tags:         make([]string, 0, 8),
	}

	// news title
	pageCrawler.OnHTML("h1", func(e *colly.HTMLElement) {
		data.Title = strings.TrimSpace(e.Text)
	})

	// news summary
	pageCrawler.OnHTML(".lead", func(e *colly.HTMLElement) {
		data.Summary = strings.TrimSpace(e.Text)
	})

	//news body
	pageCrawler.OnHTML(".story", func(e *colly.HTMLElement) {
		data.Text = strings.TrimSpace(e.Text)
	})

	// news code
	pageCrawler.OnResponse(func(r *colly.Response) {
		// fmt.Println(strings.Split(e.Text, ":")[1])
		// data.Code = strings.TrimSpace(strings.Split(e.Text, ":")[1])
		data.Code = t.NewsCode.FindString(r.Request.URL.String())
	})

	// news date and time
	pageCrawler.OnHTML(".time", func(e *colly.HTMLElement) {
		// fmt.Println(strings.Split(e.Text, ":")[1])
		data.DateTime = strings.TrimSpace(e.Text)
	})

	err := pageCrawler.Visit(url)
	return data, err
}
