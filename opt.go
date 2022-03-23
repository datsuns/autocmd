package main

import (
	"flag"
	"strings"
)

type Option struct {
	V bool
	P string
	C string
	A []string
	E []string
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
	flag.BoolVar(&ret.V, "v", false, "verbose")
	flag.StringVar(&ret.P, "p", ".", "path to watch")
	flag.Var(&excludes, "e", "exclude pattern(s)")
	flag.Parse()
	ret.C = flag.Args()[0]
	ret.A = flag.Args()[1:]
	ret.E = excludes
	Verbose = ret.V
	return ret
}
