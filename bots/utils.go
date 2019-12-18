package bots

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
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
