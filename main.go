package main

import (
	"os/exec"
	"time"

	"github.com/MShoaei/NewsMiner/bots"
)

func main() {
	// number 10 is just that, A NUMBER!
	// it can be changed but should at least be equal
	// to the number of bots running in parallel!
	exportCmds := make(chan *exec.Cmd, 10)

	go bots.BBCExtract(exportCmds)
	// go bots.YJCExtract(exportCmds)
	go bots.FarsNewsExtract(exportCmds)
	// go bots.TabnakExtract(exportCmds)
	go bots.TasnimExtract(exportCmds)

	// short sleep to make sure all the bots have started
	// successfully before closing the channel.
	time.Sleep(4 * time.Second)

	// colsing the channel is needed because the bots.Log() function
	// needs to know when to end the for loop and finish the execution.
	close(exportCmds)

	bots.Log(exportCmds)
}
