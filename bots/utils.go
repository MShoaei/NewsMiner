package bots

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/debug"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// type Done struct {
// 	Command *exec.Cmd
// }

func Log(exportCmdCh <-chan *exec.Cmd) {
	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, os.Interrupt)

	select {
	case <-sigCh:
		for command := range exportCmdCh {
			fmt.Println("exporting data")
			fmt.Println(command.String())
			if err := command.Run(); err != nil {
				log.Println(err)
			}
		}
		os.Exit(0)
	}
}

func newCrawler(archive *regexp.Regexp) *colly.Collector {
	return colly.NewCollector(
		colly.MaxDepth(1),
		colly.URLFilters(archive),
		colly.Debugger(&debug.LogDebugger{}),
	)
}
func checkNewsCode(e *colly.HTMLElement, codeRegex *regexp.Regexp, collection *mongo.Collection) string {
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
	return code.Code
}
