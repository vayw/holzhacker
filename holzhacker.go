package main

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
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

func LogDirCheck(p string) {
	err := os.MkdirAll(p, 0750)
	if err != nil {
		panic(err)
	}
}

func main() {
	var (
		logf            = flag.String("log", "error.log", "path to log file to process")
		statusDir       = flag.String("status.dir", ".local/holzhacker", "path to holzhacker status dir, if relative it will be created in $HOME")
		logtype         = flag.String("type", "access", "log type: access or error")
		showerr         = flag.Bool("show", false, "show last error for <log>")
		workers         = flag.Int("workers", 2, "number of workers")
		maintenance     = flag.String("mnt", "", "full path to maintenance file; skip check if exists")
		maintenance_num = flag.Int("mnt.num", -1, "int, which will be printed in case of maintenance skip")
		StatusFilePath  string
	)
	flag.Parse()

	var logger = log.New(os.Stderr, "holzhacker ", log.Ltime)

	file_pref := GetMD5Hash(*logf)
	if *showerr {
		logger.Print("last errors for " + *logf)
		show("/run/user/", file_pref, *logf)
		os.Exit(0)
	}

	if *maintenance != "" {
		_, m_err := os.Stat(*maintenance)
		if m_err == nil {
			logger.Print("maintenance file found! skipping this run")
			fmt.Println(*maintenance_num)
			os.Exit(0)
		}
	}

	if filepath.IsAbs(*statusDir) {
		LogDirCheck(*statusDir)
		StatusFilePath = filepath.Join(*statusDir, file_pref)
	} else {
		userinfo, _ := user.Current()
		fp := filepath.Join(userinfo.HomeDir, *statusDir)
		LogDirCheck(fp)
		StatusFilePath = filepath.Join(userinfo.HomeDir, *statusDir, file_pref)
	}

	st, err := os.OpenFile(StatusFilePath, os.O_RDWR|os.O_CREATE, 0660)
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
	go store("/run/user/", errlines, file_pref)

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
