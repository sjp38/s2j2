package irc

import (
	"bufio"
	"fmt"
	"net"
	"net/textproto"
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
	reader  *textproto.Reader
}

func NewIRC(srv, port, pass, chann, nick string) *Bot {
	return &Bot{srv, port, pass, chann, nick, nil, nil}
}

func (bot *Bot) Connect() {
	var err error
	bot.conn, err = net.Dial("tcp", bot.server+":"+bot.port)
	if err != nil {
		fmt.Printf("Error while dialing to %s.\n", bot.server+":"+bot.port)
		time.Sleep(10 * time.Second)
		bot.Connect()
	}
	fmt.Printf("Connected to %s\n", bot.server+":"+bot.port)

	fmt.Fprintf(bot.conn, "USER %s 8 * :%s\r\n", bot.nick, bot.nick)
	fmt.Fprintf(bot.conn, "PASS %s\r\n", bot.pass)
	fmt.Fprintf(bot.conn, "NICK %s\r\n", bot.nick)
	fmt.Fprintf(bot.conn, "JOIN %s\r\n", bot.channel)
	rbuf := bufio.NewReader(bot.conn)
	bot.reader = textproto.NewReader(rbuf)
}

func (bot *Bot) Close() {
	bot.conn.Close()
}

func (bot *Bot) ReadLine() (string, error) {
	return bot.reader.ReadLine()
}

func (bot *Bot) SendPrivMSG(format string, args ...interface{}) {
	privmsg_pref := "PRIVMSG " + bot.channel + " :"
	msg := fmt.Sprintf(format, args...)
	for _, line := range strings.Split(msg, "\n") {
		if line == "" {
			line = " "
		}
		fmt.Printf("[%s] SEND %s\n", time.Now(), privmsg_pref+line)
		fmt.Fprintf(bot.conn, privmsg_pref+line+"\r\n")
	}
}
