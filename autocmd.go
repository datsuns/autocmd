package main

import (
	"bufio"
	"fmt"
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
	logging *Log
}

func NewAutoCommand(o *Option) (*AutoCommand, error) {
	var err error
	ret := &AutoCommand{option: o}
	if ret.logging, err = NewLog(o.LogPath, Verbose); err != nil {
		return nil, err
	}
	ret.watcher = ret.genWatcher(o.WatchRoot, o.Excludes, o.Targets)
	return ret, nil
}

func (a *AutoCommand) run() {
	cancel := make(chan int)
	wg := &sync.WaitGroup{}

	wg.Add(1)
	a.watchMain(a.watcher, a.option.Command, a.option.Args, cancel, wg)
	a.logging.Log("started")

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		a.logging.Log("start reload")
		cancel <- 1
		wg.Wait()

		w := a.genWatcher(a.option.WatchRoot, a.option.Excludes, a.option.Targets)
		defer w.Close()
		a.logging.Reset()

		wg.Add(1)
		a.watchMain(w, a.option.Command, a.option.Args, cancel, wg)
	}
}

func (a *AutoCommand) watchMain(w *fsnotify.Watcher, cmd string, args []string, cancel chan int, wg *sync.WaitGroup) {
	go func() {
		a.watch(w, cmd, args, cancel)
		wg.Done()
	}()
}

func (a *AutoCommand) watch(w *fsnotify.Watcher, cmd string, args []string, cancel chan int) {
	for {
		select {
		case <-cancel:
			a.logging.Log("canceled")
			return
		case event, ok := <-w.Events:
			if !ok {
				return
			}
			if a.option.ClearLog {
				a.logging.Reset()
			}

			a.logging.RunLog(fmt.Sprintf("event:%v", event))
			if event.Op&fsnotify.Write == fsnotify.Write {
				a.logging.RunLog(fmt.Sprintf("modified file:%v", event.Name))
				execute(a.logging.Writer, cmd, args...)
			} else if event.Op&fsnotify.Remove == fsnotify.Remove {
				a.logging.RunLog(fmt.Sprintf("removed file:%v", event.Name))
				time.Sleep(delayToReadd)
				w.Add(event.Name)
				execute(a.logging.Writer, cmd, args...)
			}
		case err, ok := <-w.Errors:
			if !ok {
				return
			}
			a.logging.Log(fmt.Sprintf("error:%v", err))
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

func (a *AutoCommand) genWatcher(root string, exclueds []string, targets []string) (w *fsnotify.Watcher) {
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
				a.logging.Log("add : " + path)
				err = w.Add(path)
			}
		} else {
			a.logging.Log("add : " + path)
			err = w.Add(path)
		}
		return err
	})
	if err != nil {
		log.Fatal(err)
	}
	return w
}
