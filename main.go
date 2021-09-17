package main

import (
	"os/exec"
	"time"

	"github.com/MShoaei/NewsMiner/bots"
)

func main() {
	// channel buffer must be the same size as the number of bots
	exportCmds := make(chan *exec.Cmd, 5)

	go bots.FarsNewsExtract(exportCmds)
	go bots.ISNAExtract(exportCmds)
	go bots.TabnakExtract(exportCmds)
	go bots.TasnimExtract(exportCmds)
	go bots.YJCExtract(exportCmds)

	// short sleep to make sure all the bots have started
	// successfully before closing the channel.
	time.Sleep(4 * time.Second)

	// colsing the channel is needed because the bots.Log() function
	// needs to know when to end the for loop and finish the execution.
	close(exportCmds)

	bots.Log(exportCmds)
}
