package main

import (
	"flag"
	"os"
	"os/signal"

	"node-label-controller/config"
	"node-label-controller/controller"

	"k8s.io/klog"
)

var (
	path = flag.String("config", "config/config.json", "the path for the controller engine config")
)

func main() {
	flag.Parse()
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)

	configs, err := config.LoadConfig(*path)
	if err != nil {
		panic(err)
	}

	if configs == nil {
		panic("configs cannot be empty")
	}

	controllersEngine, err := controller.NewEngine(configs)
	if err != nil {
		panic(err)
	}

	go func() {
		select {
		case sig := <-c:
			klog.Info("got %s signal, shutdown the controller gracefully...\n", sig)
			controllersEngine.Stop()
			os.Exit(0)
		}
	}()

	controllersEngine.Start()

	select {}
}
