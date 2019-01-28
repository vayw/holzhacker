package main

import (
	"log"
	"os"
	"strconv"

	"github.com/vjeantet/grok"
)

func parseError(workernum int, loglines chan string, result chan int) {
	logger_prefix := "worker " + strconv.Itoa(workernum) + " "
	var logger = log.New(os.Stderr, logger_prefix, log.Ltime)
	logger.Print("started..")
	g, _ := grok.NewWithConfig(&grok.Config{NamedCapturesOnly: true})
	g.AddPattern("NGXTIME", `%{YEAR}/%{MONTHNUM}/%{MONTHDAY} %{HOUR}:%{MINUTE}:%{SECOND}`)
	parsepattern := `%{NGXTIME:tm} \[%{WORD:msg}\]`
	err_count := 0
	lines_count := 0
	unparsed := 0

	for line := range loglines {
		res, _ := g.ParseTyped(parsepattern, line)
		lines_count++
		switch res["msg"] {
		case "error", "crit":
			err_count++
		case nil:
			unparsed++
			logger.Print("can't parse:")
			logger.Print(line)
		}
	}
	if unparsed > 10 {
		logger.Printf("[WARN] %d lines haven't parsed")
	}
	logger.Printf("lines processed: %d\n", lines_count)
	result <- err_count
}
