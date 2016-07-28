package main

import (
	"bufio"
	"fmt"
	"net"
	"net/textproto"
	"os"
	"strconv"
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

func (bot *Bot) send_privmsg(msg string) {
	privmsg_pref := "PRIVMSG " + bot.channel + " :"
	fmt.Fprintf(bot.conn, privmsg_pref + msg + "\r\n")
}

func (bot *Bot) handle_privmsg(line string) {
	privmsg_pref := "PRIVMSG " + bot.channel + " :"
	msg := strings.Split(line, privmsg_pref)[1]
	if !strings.HasPrefix(msg, bot.nick + ": ") {
		return
	}
	msg = strings.Split(msg, bot.nick + ": ")[1]

	tokens := strings.Fields(msg)
	fmt.Printf("tokens: %s\n", tokens)
	if len(tokens) < 1 {
		return
	}
	switch tokens[0] {
	case "add":
		if len(tokens) < 3 {
			bot.send_privmsg("add should have two operands.")
		}
		oper1, err := strconv.Atoi(tokens[1])
		if err != nil {
			bot.send_privmsg("operand 1 should be integer.")
			return
		}
		oper2, err := strconv.Atoi(tokens[2])
		if err != nil {
			bot.send_privmsg("operand 2 should be integer.")
			return
		}

		bot.send_privmsg(fmt.Sprintf("Answer is %d\n", oper1 + oper2))
	case "hi", "hello":
		bot.send_privmsg("Hello, how are you? :D")
	case "bye":
		bot.send_privmsg("Good bye.  See you later ;)")
	default:
		bot.send_privmsg("Sorry, I cannot understand what you mean.")
	}

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
	privmsg_pref := "PRIVMSG " + bot.channel + " :"
	bot.send_privmsg("Hi, my name is S2J2.")
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
		} else if strings.Contains(line, privmsg_pref) {
			bot.handle_privmsg(line)
		}
	}
}
