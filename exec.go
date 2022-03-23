package main

import (
	"bufio"
	"io"
	"log"
	"os/exec"
)

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
	mylog(">cmd<", c, ">", p)
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
		mylog(">ERR: ", err)
	}
	return result
}
