package main

func main() {
	o, err := parse_option()
	if err != nil {
		panic(err)
	}
	a, _ := NewAutoCommand(o)
	a.run()
}
