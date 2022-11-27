package main

var (
	Verbose = false
)

func main() {
	o, err := parse_option()
	if err != nil {
		panic(err)
	}
	a, e := NewAutoCommand(o)
	if e != nil {
		panic(e)
	}
	a.run()
}
