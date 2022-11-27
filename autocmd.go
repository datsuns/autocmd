package main

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
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
	logf    io.Writer
}

func NewAutoCommand(o *Option) (*AutoCommand, error) {
	ret := &AutoCommand{option: o}
	if err := ret.logOpen(); err != nil {
		return nil, err
	}
	ret.watcher = ret.genWatcher(o.WatchRoot, o.Excludes, o.Targets)
	return ret, nil
}

func (a *AutoCommand) logOpen() error {
	var err error
	if len(a.option.LogPath) > 0 {
		if a.logf, err = os.Create(a.option.LogPath); err != nil {
			return err
		}
	} else {
		a.logf = ioutil.Discard
	}
	return nil
}

func (a *AutoCommand) log(s string) {
	log.Println(s)
	a.logf.Write([]byte(s + "\n"))
}

func (a *AutoCommand) run() {
	cancel := make(chan int)
	wg := &sync.WaitGroup{}

	wg.Add(1)
	a.watchMain(a.watcher, a.option.Command, a.option.Args, cancel, wg)
	a.log("started")

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		a.log("start reload")
		cancel <- 1
		wg.Wait()

		w := a.genWatcher(a.option.WatchRoot, a.option.Excludes, a.option.Targets)
		defer w.Close()
		a.logOpen()

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
			a.log("canceled")
			return
		case event, ok := <-w.Events:
			if !ok {
				return
			}
			a.log(fmt.Sprintf("event:%v", event))
			if event.Op&fsnotify.Write == fsnotify.Write {
				a.log(fmt.Sprintf("modified file:%v", event.Name))
				execute(a.logf, cmd, args...)
			} else if event.Op&fsnotify.Remove == fsnotify.Remove {
				a.log(fmt.Sprintf("removed file:%v", event.Name))
				time.Sleep(delayToReadd)
				w.Add(event.Name)
				execute(a.logf, cmd, args...)
			}
		case err, ok := <-w.Errors:
			if !ok {
				return
			}
			a.log(fmt.Sprintf("error:%v", err))
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
				a.log("add : " + path)
				err = w.Add(path)
			}
		} else {
			a.log("add : " + path)
			err = w.Add(path)
		}
		return err
	})
	if err != nil {
		log.Fatal(err)
	}
	return w
}
