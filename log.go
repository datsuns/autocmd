package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
)

type Log struct {
	Verbose  bool
	FilePath string
	File     *os.File
	Writer   io.Writer
}

func NewLog(fpath string, verbose bool) (ret *Log, err error) {
	ret = &Log{FilePath: fpath, Verbose: verbose}
	err = nil
	if len(ret.FilePath) > 0 {
		if ret.File, err = os.Create(fpath); err != nil {
			return nil, err
		}
		ret.Writer = ret.File
	} else {
		ret.Writer = ioutil.Discard
	}
	return ret, nil
}

func (l *Log) Reset() {
	if len(l.FilePath) > 0 {
		l.File.Close()
		l.File, _ = os.Create(l.FilePath)
		l.Writer = l.File
	}
}

func (l *Log) RunLog(v ...interface{}) {
	if l.Verbose == false {
		return
	}
	log.Println(v...)
	t := fmt.Sprint(v...)
	l.Writer.Write([]byte(t + "\n"))
}

func (l *Log) Log(v ...interface{}) {
	log.Println(v...)
	t := fmt.Sprint(v...)
	l.Writer.Write([]byte(t + "\n"))
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
