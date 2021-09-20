package bots

import (
	"regexp"
	"sync"

	"github.com/MShoaei/NewsMiner/database"
	"github.com/gocolly/colly/v2"
	"go.mongodb.org/mongo-driver/mongo"
)

type NewsAgency string

const (
	FarsnewsAgency NewsAgency = "Farsnews"
	ISNAAgency     NewsAgency = "ISNA"
	TabnakAgency   NewsAgency = "Tabnak"
	TasnimAgency   NewsAgency = "Tasnim"
	YJCAgency      NewsAgency = "YJC"
)

type Extractor interface {
	Extract(pages int)
}

type bot struct {
	wg *sync.WaitGroup

	NewsPage   *regexp.Regexp
	NewsCode   *regexp.Regexp
	Threads    int
	DB         *database.DB
	Collection *mongo.Collection

	ArchiveURL     string
	ArchiveCrawler *colly.Collector
}
