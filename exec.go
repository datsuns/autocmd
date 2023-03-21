package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os/exec"
)

func printer(logf io.Writer, reader io.ReadCloser, done chan bool) string {
	ret := ""
	scanner := bufio.NewScanner(reader)
	go func() {
		for scanner.Scan() {
			t := scanner.Text()
			logf.Write([]byte(t + "\n"))
			log.Println(t)
			ret += scanner.Text() + "\n"
		}
		done <- true
	}()
	return ret
}

func execute(logf io.Writer, c string, p ...string) (result string) {
	log.Println(">cmd<", c, ">", p)
	logf.Write([]byte(fmt.Sprintf(">cmd<%v> %v", c, p)))
	cmd := exec.Command(c, p...)
	reader_o, _ := cmd.StdoutPipe()
	reader_e, _ := cmd.StderrPipe()
	done_o := make(chan bool)
	done_e := make(chan bool)
	printer(logf, reader_o, done_o)
	printer(logf, reader_e, done_e)
	cmd.Start()
	<-done_o
	<-done_e
	err := cmd.Wait()
	if err != nil {
		log.Println(">ERR: ", err)
		logf.Write([]byte(">ERR: " + err.Error() + "\n"))
	}
	return result
}
