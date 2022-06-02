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

func build_pattern(list []string) []*regexp.Regexp {
	ret := []*regexp.Regexp{}
	if len(list) == 0 {
		return ret
	}

	for _, s := range list {
		ret = append(ret, regexp.MustCompile(s))
	}
	return ret
}

func find_pattern(matchers []*regexp.Regexp, s string) bool {
	for _, r := range matchers {
		if r.MatchString(s) {
			return true
		}
	}
	return false
}

func gen_watcher(root string, exclueds []string, targets []string) (w *fsnotify.Watcher) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	tgt := build_pattern(targets)
	target_specified := len(tgt) > 0

	ex := []*regexp.Regexp{}
	if !target_specified {
		ex = build_pattern(exclueds)
	}

	err = filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}
		if find_pattern(ex, path) {
			return nil
		}
		if target_specified {
			if find_pattern(tgt, path) {
				runlog("add : ", path)
				err = w.Add(path)
			}
		} else {
			runlog("add : ", path)
			err = w.Add(path)
		}
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
	o, err := parse_option()
	if err != nil {
		panic(err)
	}
	w := gen_watcher(o.WatchRoot, o.Excludes, o.Targets)
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
		w := gen_watcher(o.WatchRoot, o.Excludes, o.Targets)
		defer w.Close()

		wg.Add(1)
		watch_main(w, o.Command, o.Args, cancel, wg)
	}
}
