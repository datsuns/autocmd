package main

import (
	"flag"
	"strings"
)

type Option struct {
	Verbose   bool
	WatchRoot string
	Command   string
	Args      []string
	Excludes  []string
}

type arrayFlags []string

func (a *arrayFlags) String() string {
	return strings.Join(*a, ",")
}

func (a *arrayFlags) Set(value string) error {
	*a = append(*a, value)
	return nil
}

func parse_option() (ret *Option) {
	var excludes arrayFlags
	ret = &Option{}
	flag.BoolVar(&ret.Verbose, "v", false, "verbose")
	flag.StringVar(&ret.WatchRoot, "p", ".", "path to watch")
	flag.Var(&excludes, "e", "exclude pattern(s)")
	flag.Parse()
	ret.Command = flag.Args()[0]
	ret.Args = flag.Args()[1:]
	ret.Excludes = excludes
	Verbose = ret.Verbose
	return ret
}
