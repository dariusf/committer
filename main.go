package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"

	"github.com/vadiminshakov/committer/config"
	"github.com/vadiminshakov/committer/hooks"
	"github.com/vadiminshakov/committer/server"
)

func main1() {
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
	s.Stop()
}

func main() {
	if _, set := os.LookupEnv("COORDINATOR"); set {
		f, err := os.Create(fmt.Sprintf("profile.prof"))
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		runtime.SetCPUProfileRate(100000)
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}

		// c := make(chan os.Signal, 1)
		// signal.Notify(c, os.Interrupt)
		// go func() {
		// 	for range c {
		// 		os.Exit(0)
		// 	}
		// }()
	}

	main1()
	pprof.StopCPUProfile()
}
