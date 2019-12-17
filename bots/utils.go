package bots

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func Log(cmd *exec.Cmd, done <-chan struct{}) {
	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	select {
	case <-sigCh:
	case <-done:
		fmt.Println("exporting data")
		fmt.Println(cmd.String())
		if err := cmd.Run(); err != nil {
			log.Println(err)
			os.Exit(1)
		}
		os.Exit(0)
	}
}
