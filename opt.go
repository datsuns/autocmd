package main

import (
	"errors"
	"flag"
	"strings"
)

type Option struct {
	Verbose   bool
	LogPath   string
	WatchRoot string
	Command   string
	Args      []string
	Excludes  []string
	Targets   []string
}

type arrayFlags []string

func (a *arrayFlags) String() string {
	return strings.Join(*a, ",")
}

func (a *arrayFlags) Set(value string) error {
	*a = append(*a, value)
	return nil
}

func parse_option() (ret *Option, err error) {
	var excludes arrayFlags
	var targets arrayFlags

	ret = &Option{}
	flag.BoolVar(&ret.Verbose, "v", false, "verbose")
	flag.StringVar(&ret.LogPath, "l", "", "path to log file")
	flag.StringVar(&ret.WatchRoot, "p", ".", "path to watch")
	flag.Var(&excludes, "e", "exclude pattern(s). ignored if target pattern specified")
	flag.Var(&targets, "t", "target pattern(s)")
	flag.Parse()

	switch flag.NArg() {
	case 0:
		return nil, errors.New("command must be set")
	case 1:
		ret.Command = flag.Args()[0]
	default:
		ret.Command = flag.Args()[0]
		ret.Args = flag.Args()[1:]
	}
	ret.Excludes = excludes
	ret.Targets = targets
	Verbose = ret.Verbose
	return ret, nil
}
