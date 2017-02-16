package main

import (
	"os"
	"strings"

	"./pkg/irc"
)

func main() {
	c := irc.NewIRC(os.Args[1],
		os.Args[2],
		os.Args[3],
		os.Args[4],
		os.Args[5])

	c.Connect()
	defer c.Close()
	c.SendPrivMSG(strings.Join(os.Args[6:], " "))
}
