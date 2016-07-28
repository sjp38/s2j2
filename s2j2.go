package main

import (
	"bufio"
	"fmt"
	"net"
	"net/textproto"
	"os"
	"strings"
	"time"
)

type Bot struct {
	server  string
	port    string
	pass    string
	channel string
	nick    string
	conn    net.Conn
}

func (bot *Bot) connect_irc() {
	var err error
	bot.conn, err = net.Dial("tcp", bot.server+":"+bot.port)
	if err != nil {
		fmt.Printf("Error while dialing to %s.\n", bot.server+":"+bot.port)
		time.Sleep(10 * time.Second)
		bot.connect_irc()
	}
	fmt.Printf("Connected to %s\n", bot.server+":"+bot.port)
}

func main() {
	fmt.Printf("%s\n", os.Args)
	if len(os.Args) < 6 {
		fmt.Printf("usage: s2j2 <server> <port> <pass> <channel> <nick>\n")
		os.Exit(1)
	}
	bot := &Bot{
		server:  os.Args[1],
		port:    os.Args[2],
		pass:    os.Args[3],
		channel: os.Args[4],
		nick:    os.Args[5],
	}

	bot.connect_irc()
	fmt.Fprintf(bot.conn, "USER %s 8 * :%s\r\n", bot.nick, bot.nick)
	fmt.Fprintf(bot.conn, "PASS %s\r\n", bot.pass)
	fmt.Fprintf(bot.conn, "NICK %s\r\n", bot.nick)
	fmt.Fprintf(bot.conn, "JOIN %s\r\n", bot.channel)
	defer bot.conn.Close()
	rbuf := bufio.NewReader(bot.conn)
	txtin := textproto.NewReader(rbuf)
	fmt.Fprintf(bot.conn, "PRIVMSG " + bot.channel + " :" + "Hello, my name is S2J2\r\n")
	for {
		line, err := txtin.ReadLine()
		if err != nil {
			fmt.Printf("error while reading input!\n")
			break
		}
		fmt.Printf("read %s\n", line)
		if strings.Contains(line, "PING ") {
			fmt.Printf(" has PING: %s\n", line)
			pongdata := strings.Split(line, "PING ")
			fmt.Printf("%s", pongdata)
			fmt.Fprintf(bot.conn, "PONG %s\r\n", pongdata[1])
		}
	}
}
