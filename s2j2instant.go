package main

import (
	"os"
	"strings"

	"./pkg/irc"
)

func main() {
	bot := irc.NewIRC(os.Args[1],
		os.Args[2],
		os.Args[3],
		os.Args[4],
		os.Args[5])

	bot.Connect()
	defer bot.Close()
	bot.SendPrivMSG(strings.Join(os.Args[6:], " "))
}
