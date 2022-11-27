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
	// s/w somtimes write a file as "delete and new-write".
	// we should re-add to fsnotify when watching file removed
	delayToReadd = time.Millisecond * 200
)

type AutoCommand struct {
	watcher *fsnotify.Watcher
	option  *Option
}

func NewAutoCommand(o *Option) *AutoCommand {
	ret := &AutoCommand{option: o}
	ret.watcher = genWatcher(o.WatchRoot, o.Excludes, o.Targets)
	return ret
}

func (a *AutoCommand) run() {
	cancel := make(chan int)
	wg := &sync.WaitGroup{}

	wg.Add(1)
	watchMain(a.watcher, a.option.Command, a.option.Args, cancel, wg)
	mylog("started")

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		mylog("start reload")
		cancel <- 1
		wg.Wait()

		w := genWatcher(a.option.WatchRoot, a.option.Excludes, a.option.Targets)
		defer w.Close()

		wg.Add(1)
		watchMain(w, a.option.Command, a.option.Args, cancel, wg)
	}
}

func watchMain(w *fsnotify.Watcher, cmd string, args []string, cancel chan int, wg *sync.WaitGroup) {
	go func() {
		watch(w, cmd, args, cancel)
		wg.Done()
	}()
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

func findPattern(matchers []*regexp.Regexp, s string) bool {
	for _, r := range matchers {
		if r.MatchString(s) {
			return true
		}
	}
	return false
}

func buildPattern(list []string) []*regexp.Regexp {
	ret := []*regexp.Regexp{}
	if len(list) == 0 {
		return ret
	}

	for _, s := range list {
		ret = append(ret, regexp.MustCompile(s))
	}
	return ret
}

func genWatcher(root string, exclueds []string, targets []string) (w *fsnotify.Watcher) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	tgt := buildPattern(targets)
	target_specified := len(tgt) > 0

	ex := []*regexp.Regexp{}
	if !target_specified {
		ex = buildPattern(exclueds)
	}

	err = filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}
		if findPattern(ex, path) {
			return nil
		}
		if target_specified {
			if findPattern(tgt, path) {
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
