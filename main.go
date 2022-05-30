package main

import (
	"bufio"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const (
	// s/w somtimes write by "delete and new-write".
	// we should re-add to fsnotify when watching file removed
	delayToReadd = time.Millisecond * 200
)

var (
	Verbose = false
)

func gen_watcher(root string, exclueds []string) (w *fsnotify.Watcher) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	ex := []*regexp.Regexp{}
	for _, e := range exclueds {
		ex = append(ex, regexp.MustCompile(e))
	}

	err = filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}
		for _, r := range ex {
			if r.MatchString(path) {
				return nil
			}
		}
		runlog("add : ", path)
		err = w.Add(path)
		return err
	})
	if err != nil {
		log.Fatal(err)
	}
	return w
}

func watch(w *fsnotify.Watcher, cmd string, args []string, cancel chan int) {
	for {
		select {
		case <-cancel:
			mylog("canceled")
			return
		case event, ok := <-w.Events:
			if !ok {
				return
			}
			runlog("event:", event)
			if event.Op&fsnotify.Write == fsnotify.Write {
				runlog("modified file:", event.Name)
				execute(cmd, args...)
			} else if event.Op&fsnotify.Remove == fsnotify.Remove {
				runlog("removed file:", event.Name)
				time.Sleep(delayToReadd)
				w.Add(event.Name)
				execute(cmd, args...)
			}
		case err, ok := <-w.Errors:
			if !ok {
				return
			}
			mylog("error:", err)
		}
	}
}

func watch_main(w *fsnotify.Watcher, cmd string, args []string, cancel chan int, wg *sync.WaitGroup) {
	go func() {
		watch(w, cmd, args, cancel)
		wg.Done()
	}()
}

func main() {
	o := parse_option()
	w := gen_watcher(o.WatchRoot, o.Excludes)
	defer w.Close()

	cancel := make(chan int)
	wg := &sync.WaitGroup{}

	wg.Add(1)
	watch_main(w, o.Command, o.Args, cancel, wg)
	mylog("started")

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		mylog("start reload")
		cancel <- 1
		wg.Wait()

		w.Close()
		w := gen_watcher(o.WatchRoot, o.Excludes)
		defer w.Close()

		wg.Add(1)
		watch_main(w, o.Command, o.Args, cancel, wg)
	}
}
