package main

var (
	Verbose = false
)

func main() {
	o, err := parse_option()
	if err != nil {
		panic(err)
	}
	a := NewAutoCommand(o)
	a.run()
}
