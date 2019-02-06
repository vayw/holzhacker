package main

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
)

// Reading files requires checking most calls for errors.
// This helper will streamline our error checks below.
func check(e error) {
	if e != nil {
		panic(e)
	}
}

func GetMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func main() {
	var (
		logf    = flag.String("log", "error.log", "path to log file to process")
		status  = flag.String("status", "holzhacker.status", "path to holzhacker status file")
		logtype = flag.String("type", "access", "log type: access or error")
		showerr = flag.Bool("show", false, "show last error for <log>")
		workers = flag.Int("workers", 2, "number of workers")
	)
	flag.Parse()

	var logger = log.New(os.Stderr, "holzhacker ", log.Ltime)

	err_file_pref := GetMD5Hash(*logf)
	if *showerr {
		logger.Print("last errors for " + *logf)
		show("/run/user/", err_file_pref, *logf)
		os.Exit(0)
	}

	st, err := os.OpenFile(*status, os.O_RDWR|os.O_CREATE, 0660)
	check(err)
	status_r := bufio.NewReader(st)
	offset_str, err := status_r.ReadString('\n')
	var offset int64
	if offset_str == "" {
		offset = 0
	} else {
		offset_str = offset_str[:len(offset_str)-1]
		offset, err = strconv.ParseInt(offset_str, 10, 64)
		check(err)
	}

	f, err := os.Open(*logf)
	check(err)
	defer f.Close()

	f.Seek(offset, 0)

	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)

	// define channels
	loglines := make(chan string, 200)
	results := make(chan int, *workers)
	errlines := make(chan string, 200)

	// wait groups
	wg := &sync.WaitGroup{}

	switch *logtype {
	case "access":
		for i := 0; i < *workers; i++ {
			wg.Add(1)
			go parseAccess(i, loglines, results, errlines, wg)
		}
	case "error":
		for i := 0; i < *workers; i++ {
			wg.Add(1)
			go parseError(i, loglines, results, errlines, wg)
		}
	default:
		panic("PANIC! unknown log type: " + *logtype)
	}

	go concierge(wg, errlines)
	go store("/run/user/", errlines, err_file_pref)

	lines_count := 0
	for scanner.Scan() {
		cls := scanner.Text()
		loglines <- cls
		lines_count++
	}
	logger.Printf("log lines sent: %d", lines_count)
	close(loglines)

	err_count := 0
	for i := 0; i < *workers; i++ {
		c := <-results
		err_count += c
	}

	if err := scanner.Err(); err != nil {
		logger.Printf("reading file: %s", err)
	}

	currentPosition, err := f.Seek(0, 1)
	st.Truncate(0)
	st.Seek(0, 0)
	st.WriteString(fmt.Sprintf("%d\n", currentPosition))

	logger.Print("errors:")
	fmt.Println(err_count)
}
