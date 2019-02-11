package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/user"
	"sync"
)

func store(logdir string, errchan chan string, pref string) {
	userinfo, _ := user.Current()
	uid := userinfo.Uid
	f, err := os.Create(logdir + uid + "/holzhacker." + pref)
	check(err)
	defer f.Close()
	for line := range errchan {
		_, err := f.WriteString(line + "\n")
		if err != nil {
			panic(err)
		}
	}
}

func concierge(wg *sync.WaitGroup, ch chan string) {
	wg.Wait()
	close(ch)
}

func show(logdir string, pref string, logname string) {
	var logger = log.New(os.Stderr, "holzhacker ", log.Ltime)
	userinfo, _ := user.Current()
	uid := userinfo.Uid
	f, err := os.Open(logdir + uid + "/holzhacker.lines." + pref)
	if err != nil {
		logger.Fatal("can't read log for: " + logname)
	}
	check(err)
	defer f.Close()

	f.Seek(0, 0)

	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		cls := scanner.Text()
		fmt.Println(cls)
	}
}
