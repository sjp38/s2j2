package main

import (
	"bufio"
	"fmt"
	"net"
	"net/textproto"
	"os"
	"os/exec"
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
	fmt.Fprintf(bot.conn, privmsg_pref+msg+"\r\n")
}

var poll_question string
var poll_selections []string
var poll_results map[int][]string
var poll_owner string

func (bot *Bot) do_poll(peername string, tokens []string) {
	command := tokens[1]
	switch command {
	case "question":
		poll_question = strings.Join(tokens[2:], " ")
		poll_owner = peername
	case "selections":
		arg := strings.Join(tokens[2:], " ")
		fmt.Printf("arg: %s\n")
		poll_selections = []string{}
		for i, selection := range strings.Split(arg, ",") {
			poll_selections = append(poll_selections, fmt.Sprintf("%d. %s", i, strings.Trim(selection, " ")))
		}
		poll_owner = peername
	case "notify":
		if poll_question == "" {
			bot.send_privmsg("No poll is going on now.")
			return
		}

		bot.send_privmsg(" ")
		bot.send_privmsg("Current Poll")
		bot.send_privmsg("============")
		bot.send_privmsg(" ")
		bot.send_privmsg(fmt.Sprintf("    Owner: %s", poll_owner))
		bot.send_privmsg(" ")
		bot.send_privmsg("Question")
		bot.send_privmsg("--------")
		bot.send_privmsg(" ")
		bot.send_privmsg(poll_question)
		bot.send_privmsg(" ")
		bot.send_privmsg("Selections")
		bot.send_privmsg("--------")
		for _, selection := range poll_selections {
			bot.send_privmsg("  " + selection)
		}
		bot.send_privmsg(" ")
	case "vote":
		if poll_question == "" {
			bot.send_privmsg("No poll is going on now.")
			return
		}

		selection, err := strconv.Atoi(tokens[2])
		if err != nil {
			bot.send_privmsg("Selection should be integer.")
			return
		}
		if selection < 0 || selection >= len(poll_selections) {
			bot.send_privmsg(fmt.Sprintf("Selection should be >=0, <%d",
				len(poll_selections)))
			return
		}
		for _, name := range poll_results[selection] {
			if peername == name {
				bot.send_privmsg(
					fmt.Sprintf("%s, " +
					"you already voted to the selection.",
					peername))
				return
			}
		}

		poll_results[selection] = append(poll_results[selection], peername)
	case "vote_cancle":
		if poll_question == "" {
			bot.send_privmsg("No poll is going on now.")
			return
		}

		for i, people := range poll_results {
			new_people := []string{}
			for _, name := range people {
				if name == peername {
					continue
				}
				new_people = append(new_people, name)
			}
			poll_results[i] = new_people
		}
	case "result":
		if poll_question == "" {
			bot.send_privmsg("No poll is going on now.")
			return
		}
		bot.send_privmsg("[Current result is...]")
		for i, selection := range poll_selections {
			people := poll_results[i]
			bot.send_privmsg(
				fmt.Sprintf("%s: %d (%s)",
					selection, len(people), people))
		}
	case "cleanup_result":
		if peername != poll_owner {
			bot.send_privmsg("Only owner can cleanup result")
			return
		}
		poll_results = map[int][]string{}
	case "finish":
		if peername != poll_owner {
			bot.send_privmsg("Only owner can finish poll")
			return
		}
		poll_question = ""
		poll_results = make(map[int][]string)
	case "help":
		bot.send_privmsg("Usage: poll <command> [arg...]")
		bot.send_privmsg("  commands: question, selections, notify, vote, vote_cancle, result, cleanup_result, finish, help")
		bot.send_privmsg(" NOTE:")
		bot.send_privmsg(" selections argument should be seperated by comma")
		bot.send_privmsg(" vote argument should be integer")
	}
}

func (bot *Bot) handle_privmsg(line string) {
	peername := ""
	if strings.HasPrefix(line, ":") {
		tokens := strings.Split(line, "!")
		if len(tokens) >= 2 {
			peername = tokens[0][1:]
		}
	}

	privmsg_pref := "PRIVMSG " + bot.channel + " :"
	msg := strings.Split(line, privmsg_pref)[1]
	if !strings.HasPrefix(msg, bot.nick+": ") {
		return
	}
	if peername == "" {
		fmt.Printf("It's privmsg but no peername...\n")
		return
	}
	msg = strings.Split(msg, bot.nick+": ")[1]

	tokens := strings.Fields(msg)
	fmt.Printf("tokens: %s\n", tokens)
	if len(tokens) < 1 {
		return
	}
	switch tokens[0] {
	case "poll":
		bot.do_poll(peername, tokens)
	case "ex":
		if len(tokens) < 2 {
			bot.send_privmsg("You forgot command.")
		}
		out, err := exec.Command("./" + tokens[1], tokens[2:]...).Output()
		if err != nil {
			fmt.Printf("error while command execution: %s\n", err)
			bot.send_privmsg("Failed to execute your command.")
		}
		sout := string(out)
		lines := strings.Split(sout, "\n")
		for _, line := range lines {
			if line == "" {
				line = " "
			}
			bot.send_privmsg(line)
		}
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

		bot.send_privmsg(fmt.Sprintf("%s: Answer is %d\n",
			peername, oper1+oper2))
	case "hi", "hello":
		bot.send_privmsg(
			fmt.Sprintf("Hello, %s. How are you? :D", peername))
	case "bye":
		bot.send_privmsg(
			fmt.Sprintf("Good bye, %s.  See you later ;)", peername))
	default:
		bot.send_privmsg(
			fmt.Sprintf("Sorry, %s. I cannot understand what you mean.", peername))
	}
}

func main() {
	fmt.Printf("%s\n", os.Args)
	if len(os.Args) < 6 {
		fmt.Printf("usage: s2j2 <server> <port> <pass> <channel> <nick>\n")
		os.Exit(1)
	}
	poll_results = make(map[int][]string)

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
			pongdata := strings.Split(line, "PING ")
			fmt.Printf("PONG %s\n", pongdata)
			fmt.Fprintf(bot.conn, "PONG %s\r\n", pongdata[1])
		} else if strings.Contains(line, privmsg_pref) {
			bot.handle_privmsg(line)
		} else if strings.Contains(line, " JOIN ") {
			peername := ""
			if strings.HasPrefix(line, ":") {
				tokens := strings.Split(line, "!")
				if len(tokens) >= 2 {
					peername = tokens[0][1:]
				}
			}

			if peername != "" && peername != bot.nick {
				bot.send_privmsg("Welcome, " + peername)
			}
		}
	}
}
