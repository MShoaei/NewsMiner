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

type YJC struct {
	bot
}

func NewYJCBot(threads int, db *database.DB, collection *mongo.Collection, wg *sync.WaitGroup) *YJC {
	yjcNewsRegex, _ := db.GetNewsPageRegex(string(YJCAgency))
	yjcCodeRegex, _ := db.GetNewsCodeRegex(string(YJCAgency))
	archiveURL, _ := db.GetArchiveURL(string(YJCAgency))
	y := &YJC{
		bot: bot{
			wg:         wg,
			NewsPage:   yjcNewsRegex,
			NewsCode:   yjcCodeRegex,
			DB:         db,
			Collection: collection,
			Threads:    threads,
			ArchiveURL: archiveURL,
		},
	}
	y.ArchiveCrawler = y.getArchiveCrawler()
	return y
}

func (y *YJC) Extract(pages int) {
	defer y.wg.Done()
	q := y.fillQueue(pages)
	q.Run(y.ArchiveCrawler)
}

func (y *YJC) fillQueue(pages int) *queue.Queue {
	now := ptime.New(time.Now())
	q, _ := queue.New(y.Threads, &queue.InMemoryQueueStorage{MaxSize: pages})

	currentPage := 0
	for currentPage < pages {
		count := y.getPageCount(now)
		for i := 1; i <= count; i++ {
			err := q.AddURL(fmt.Sprintf(y.ArchiveURL, now.Format("yyyy/MM/dd"), i))
			if err != nil {
				return q
			}
			currentPage++
		}
		now = now.AddDate(0, 0, -1)
	}
	return q
}

func (y *YJC) getPageCount(date ptime.Time) int {
	return 10
}

func (y *YJC) isNewsPage(e *colly.HTMLElement) bool {
	return y.NewsPage.MatchString(e.Request.AbsoluteURL(e.Attr("href")))
}

// YJCExtract starts a bot for https://www.yjc.news
func (y *YJC) getArchiveCrawler() *colly.Collector {
	linkExtractor := colly.NewCollector(
		colly.MaxDepth(1),
	)

	linkExtractor.OnHTML(".linear_news a[href]", func(e *colly.HTMLElement) {
		//log.Println(e.Attr("href"))

		partialURL := e.Attr("href")
		code := y.NewsCode.FindString(partialURL)
		if !y.isNewsPage(e) || y.DB.NewsWithCodeExists(y.Collection, code) {
			return
		}
		data, err := y.extractPageDetail(e.Request.AbsoluteURL(e.Attr("href")))
		if err != nil {
			return
		}
		y.DB.Save(y.Collection, data)
	})
	return linkExtractor
}

func (y *YJC) extractPageDetail(url string) (*NewsData, error) {
	pageCrawler := colly.NewCollector(
		colly.MaxDepth(1),
	)

	data := &NewsData{
		NewsAgencyID: "YJC",
		Tags:         make([]string, 0, 8),
	}

	// news title
	pageCrawler.OnHTML(".title", func(e *colly.HTMLElement) {
		data.Title = strings.TrimSpace(e.Text)
	})

	// news summary
	pageCrawler.OnHTML(".Htags_news_subtitle .news_strong", func(e *colly.HTMLElement) {
		data.Summary = strings.TrimSpace(e.Text)
	})

	//news body
	pageCrawler.OnHTML(".body", func(e *colly.HTMLElement) {
		data.Text = strings.TrimSpace(e.Text)
	})

	//news code
	pageCrawler.OnResponse(func(r *colly.Response) {
		data.Code = y.NewsCode.FindString(r.Request.URL.String())
	})

	// news date and time
	pageCrawler.OnHTML(".news_pdate_c", func(e *colly.HTMLElement) {
		data.DateTime = strings.TrimSpace(strings.Split(e.Text, ":")[1])
	})

	err := pageCrawler.Visit(url)
	return data, err
}
