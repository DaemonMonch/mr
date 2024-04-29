package main

import (
	"context"
	"flag"
	"fmt"
	"slmmzx/mr"
	"strings"

	"github.com/mattn/go-tty"
	"github.com/mattn/go-tty/ttyutil"
)

var (
	jobConfigFilePath = flag.String("f", "run.yml", "config file")
)

func main() {

	flag.Parse()
	cfg := mr.InitConfig(*jobConfigFilePath)

	tty, err := tty.Open()
	if err != nil {
		panic(err)
	}
	cfg.Output = tty.Output()

	jobManager := mr.NewJobManager(cfg)
	jobManager.StartJobs(context.Background())

	go func() {
		for {
			cmd, _ := ttyutil.ReadLine(tty)
			if cmd == "ls" {
				for _, j := range jobManager.Jobs() {
					fmt.Fprintf(tty.Output(), "> %s", j)
				}
			}

			if strings.Contains(cmd, "rs") {
				sp := strings.Split(cmd, " ")
				if len(sp) == 2 {
					jobManager.Restart(sp[1])
				}
			}
		}

	}()

	jobManager.WaitTerminate()
}
