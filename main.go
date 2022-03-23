package main

import (
	"io/fs"
	"log"
	"path/filepath"
	"regexp"
	"time"

	"github.com/fsnotify/fsnotify"
)

const (
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
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	return w
}

func watch_main(w *fsnotify.Watcher, cmd string, args []string) {
	for {
		select {
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

func main() {
	o := parse_option()
	w := gen_watcher(o.P, o.E)
	defer w.Close()

	watch_main(w, o.C, o.A)
}
