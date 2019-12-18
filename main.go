package main

import (
	"github.com/MShoaei/NewsMiner/bots"
	"os/exec"
	"time"
)

func main() {
	exportCmds := make(chan *exec.Cmd, 5)

	go bots.BBCExtract(exportCmds)
	// go bots.YJCExtract(exportCmds)
	go bots.FarsNewsExtract(exportCmds)
	// go bots.TabnakExtract(exportCmds)
	go bots.TasnimExtract(exportCmds)

	time.Sleep(4 * time.Second)
	close(exportCmds)
	
	bots.Log(exportCmds)
}
