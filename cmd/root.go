package cmd

import (
	"fmt"
	"os"
	"sync"

	"github.com/MShoaei/NewsMiner/bots"
	"github.com/MShoaei/NewsMiner/database"
	"github.com/spf13/cobra"
)

var rootCmd = newRootCommand()

func newRootCommand() *cobra.Command {
	var (
		dbName     string
		connString string
		threads    int
		pages      int
		farsnews   bool
		isna       bool
		tabnak     bool
		tasnim     bool
		yjc        bool
	)
	cmd := &cobra.Command{
		Use: "crawler",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := database.NewDB(dbName, connString)
			if err != nil {
				return err
			}
			c := db.CreateCollection("News")

			wg := sync.WaitGroup{}
			if farsnews {
				wg.Add(1)
				bot := bots.NewFarsnewsBot(threads, db, c, &wg)
				go bot.Extract(pages)
			}
			if isna {
				wg.Add(1)
				bot := bots.NewISNABot(threads, db, c, &wg)
				go bot.Extract(pages)
			}
			if tabnak {
				wg.Add(1)
				bot := bots.NewTabnakBot(threads, db, c, &wg)
				go bot.Extract(pages)
			}
			if tasnim {
				wg.Add(1)
				bot := bots.NewTasnimBot(threads, db, c, &wg)
				go bot.Extract(pages)
			}
			if yjc {
				wg.Add(1)
				bot := bots.NewYJCBot(threads, db, c, &wg)
				go bot.Extract(pages)
			}

			wg.Wait()
			return nil
		},
	}

	f := cmd.PersistentFlags()
	f.SortFlags = true

	f.StringVarP(&dbName, "db", "d", "", "Database name")
	cmd.MarkPersistentFlagRequired("db")
	f.StringVarP(&connString, "conn", "c", "", "Database connection string")
	cmd.MarkPersistentFlagRequired("conn")

	f = cmd.Flags()
	f.SortFlags = true

	f.IntVarP(&threads, "threads", "t", 4, "Number of threads")
	f.IntVarP(&pages, "pages", "p", 10, "Number of archive pages. in almost all cases there are more than one news per page")
	f.BoolVar(&farsnews, "farsnews", false, "Farsnews")
	f.BoolVar(&isna, "isna", false, "ISNA")
	f.BoolVar(&tabnak, "tabnak", false, "Tabnak")
	f.BoolVar(&tasnim, "tasnim", false, "Tasnim")
	f.BoolVar(&yjc, "yjc", false, "YJC")

	return cmd
}

// Execute executes the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
