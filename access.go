package main

import (
	"log"
	"os"
	"strconv"

	"github.com/vjeantet/grok"
)

func parseAccess(workernum int, loglines chan string, result chan int) {
	logger_prefix := "worker " + strconv.Itoa(workernum) + " "
	var logger = log.New(os.Stderr, logger_prefix, log.Ltime)
	logger.Print("started..")
	g, _ := grok.NewWithConfig(&grok.Config{NamedCapturesOnly: true})
	g.AddPattern("NGXTIME", `%{YEAR}/%{MONTHNUM}/%{MONTHDAY} %{HOUR}:%{MINUTE}:%{SECOND}`)
	g.AddPattern("BEFORECODE", `%{IP}.*`)
	parsepattern := `%{BEFORECODE:before}HTTP/\d\.\d"\|%{NUMBER:response_code:int}|.\d.*`
	err_count := 0
	lines_count := 0
	unparsed := 0

	for line := range loglines {
		res, _ := g.ParseTyped(parsepattern, line)
		lines_count++
		switch {
		case res["response_code"] == nil:
			unparsed++
			logger.Print("can't parse:")
			logger.Print(line)
		default:
			if (res["response_code"]).(int) >= 500 {
				err_count++
			}
		}
	}

	if unparsed > 10 {
		logger.Printf("[WARN] %d lines haven't parsed", unparsed)
	}
	logger.Printf("lines processed: %d\n", lines_count)
	result <- err_count
}
