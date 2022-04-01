package main

import "log"

func runlog(v ...interface{}) {
	if Verbose == false {
		return
	}
	log.Println(v...)
}

func mylog(v ...interface{}) {
	log.Println(v...)
}
