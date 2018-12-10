package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/vjeantet/grok"
)

// Reading files requires checking most calls for errors.
// This helper will streamline our error checks below.
func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	var log = flag.String("log", "error.log", "path to log file to process")
	var status = flag.String("status", "holzhacker.status", "path to holzhacker status file")
	flag.Parse()

	g, _ := grok.NewWithConfig(&grok.Config{NamedCapturesOnly: true})
	g.AddPattern("NGXTIME", `%{YEAR}/%{MONTHNUM}/%{MONTHDAY} %{HOUR}:%{MINUTE}:%{SECOND}`)

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

	f, err := os.Open(*log)
	check(err)
	defer f.Close()

	f.Seek(offset, 0)

	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)

	var err_count = 0
	for scanner.Scan() {
		result, _ := g.Parse(`%{NGXTIME:tm} \[%{WORD:msg}\]`, scanner.Text())
		switch result["msg"] {
		case "error", "crit":
			err_count += 1
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading file:", err)
	}

	currentPosition, err := f.Seek(0, 1)
	st.Truncate(0)
	st.Seek(0, 0)
	st.WriteString(fmt.Sprintf("%d\n", currentPosition))

	fmt.Println(err_count)
}
