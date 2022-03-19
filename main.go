package main

import (
	"bufio"
	"flag"
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

type Option struct {
	p string
	c string
	a []string
}

func parse_option() (ret *Option) {
	ret = &Option{}
	flag.StringVar(&ret.p, "p", ".", "path to watch")
	flag.Parse()
	ret.c = flag.Args()[0]
	ret.a = flag.Args()[1:]
	return ret
}

func execute(c string, p ...string) (result string) {
	cmd := exec.Command(c, p...)
	reader, _ := cmd.StdoutPipe()
	scanner := bufio.NewScanner(reader)
	done := make(chan bool)
	go func() {
		for scanner.Scan() {
			log.Printf(scanner.Text())
			result += scanner.Text() + "\n"
		}
		done <- true
	}()
	cmd.Start()
	<-done
	err := cmd.Wait()
	if err != nil {
		log.Println("ERR: ", err)
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
		log.Println("add : ", path)
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
			log.Println("event:", event)
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Println("modified file:", event.Name)
				execute(cmd, args...)
			} else if event.Op&fsnotify.Remove == fsnotify.Remove {
				log.Println("removed file:", event.Name)
				time.Sleep(delayToReadd)
				w.Add(event.Name)
				execute(cmd, args...)
			}
		case err, ok := <-w.Errors:
			if !ok {
				return
			}
			log.Println("error:", err)
		}
	}
}

func main() {
	o := parse_option()
	w := gen_watcher(o.p)
	defer w.Close()

	watch_main(w, o.c, o.a)
}
