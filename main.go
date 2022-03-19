package main

import (
	"bufio"
	"flag"
	"io"
	"io/fs"
	"log"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

const (
	delayToReadd = time.Millisecond * 200
)

var (
	Verbose = false
)

type Option struct {
	v bool
	p string
	c string
	a []string
}

func runlog(v ...interface{}) {
	if Verbose == false {
		return
	}
	log.Println(v...)
}

func mylog(v ...interface{}) {
	log.Println(v...)
}

func parse_option() (ret *Option) {
	ret = &Option{}
	flag.BoolVar(&ret.v, "v", false, "verbose")
	flag.StringVar(&ret.p, "p", ".", "path to watch")
	flag.Parse()
	ret.c = flag.Args()[0]
	ret.a = flag.Args()[1:]
	Verbose = ret.v
	return ret
}

func printer(reader io.ReadCloser, done chan bool) string {
	ret := ""
	scanner := bufio.NewScanner(reader)
	go func() {
		for scanner.Scan() {
			log.Printf(scanner.Text())
			ret += scanner.Text() + "\n"
		}
		done <- true
	}()
	return ret
}

func execute(c string, p ...string) (result string) {
	cmd := exec.Command(c, p...)
	reader_o, _ := cmd.StdoutPipe()
	reader_e, _ := cmd.StderrPipe()
	done_o := make(chan bool)
	done_e := make(chan bool)
	printer(reader_o, done_o)
	printer(reader_e, done_e)
	cmd.Start()
	<-done_o
	<-done_e
	err := cmd.Wait()
	if err != nil {
		mylog("ERR: ", err)
	}
	return result
}

func gen_watcher(root string) (w *fsnotify.Watcher) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	err = filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
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
			mylog("event:", event)
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
	w := gen_watcher(o.p)
	defer w.Close()

	watch_main(w, o.c, o.a)
}
