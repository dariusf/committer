package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/vadiminshakov/committer/config"
	"github.com/vadiminshakov/committer/hooks"
	"github.com/vadiminshakov/committer/server"
)

func main() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	conf := config.Get()
	hooks, err := hooks.Get()
	if err != nil {
		panic(err)
	}
	s, err := server.NewCommitServer(conf, hooks...)
	if err != nil {
		panic(err)
	}

	s.Run(server.WhiteListChecker)
	<-ch
	// if s.MonitorC != nil {
	// 	s.MonitorC.PrintLog()
	// }
	// if s.MonitorP != nil {
	// 	s.MonitorP.PrintLog()
	// }
	s.Stop()
}
