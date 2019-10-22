package main

import (
	"github.com/gocolly/colly"
)

func main() {
	c := colly.NewCollector()
	c.OnHTML("", func(e *colly.HTMLElement) {

	})
	c.Visit("")
}
