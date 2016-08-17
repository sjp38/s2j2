package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/smtp"
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

func (bot *Bot) send_privmsg(format string, args ...interface{}) {
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

var poll_question string
var poll_selections []string
var poll_results map[int][]string
var poll_owner string

func (bot *Bot) do_poll(peername string, tokens []string) {
	command := tokens[1]
	switch command {
	case "question":
		if poll_question != "" && peername != poll_owner {
			bot.send_privmsg("Another poll is ongoing yet.")
			return
		}
		poll_question = strings.Join(tokens[2:], " ")
		poll_owner = peername
	case "selections":
		arg := strings.Join(tokens[2:], " ")
		poll_selections = []string{}
		for i, selection := range strings.Split(arg, ",") {
			poll_selections = append(poll_selections,
				fmt.Sprintf("%d. %s", i,
					strings.Trim(selection, " ")))
		}
		poll_owner = peername
	case "notify":
		if poll_question == "" {
			bot.send_privmsg("No poll is going on now.")
			return
		}

		msg := fmt.Sprintf("\n"+
			"Current Poll\n"+
			"============\n\n"+
			"    Owner: %s\n\n"+
			"Question\n"+
			"--------\n\n"+
			poll_question+"\n\n\n"+
			"Selections\n"+
			"----------\n\n", poll_owner)
		for _, selection := range poll_selections {
			msg += "  " + selection + "\n"
		}
		bot.send_privmsg(msg)
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
			bot.send_privmsg("Selection should be >=0, <%d",
				len(poll_selections))
			return
		}
		for _, name := range poll_results[selection] {
			if peername == name {
				bot.send_privmsg("%s, "+
					"you already voted to the selection.",
					peername)
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
			bot.send_privmsg("%s: %d (%s)",
				selection, len(people), people)
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
		poll_results = map[int][]string{}
	case "help":
		bot.send_privmsg("Usage: poll <command> [arg...]")
		bot.send_privmsg("  commands: question, selections, notify, ",
			"vote, vote_cancle, result, ",
			"cleanup_result, finish, help")
		bot.send_privmsg(" NOTE:")
		bot.send_privmsg(" selections argument should be seperated by comma")
		bot.send_privmsg(" vote argument should be integer")
	}
}

func fetchHtmlTitle(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		return "Failed to GET the url"
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return "Failed to get body of response"
	}

	html_src := string(body)
	// TODO: do parse correctly
	titleStart := strings.Index(html_src, "<title>") + len("<title>")
	titleEnd := strings.Index(html_src, "</title>")
	if titleStart == -1 || titleEnd == -1 || titleEnd < titleStart {
		return "Failed to find title"
	}
	title := html.UnescapeString(html_src[titleStart:titleEnd])
	if titleEnd-titleStart > 720 {
		title = title[0:720]
	}
	return title
}

type account struct {
	Username string
	Password string
}

const gmailAccountFile = "gmailinfo"

var gmailAccount account

func read_gmailinfo() {
	c, err := ioutil.ReadFile(gmailAccountFile)
	if err != nil {
		fmt.Printf("failed to read mail info file: %s\n", err)
		return
	}
	if err := json.Unmarshal(c, &gmailAccount); err != nil {
		fmt.Printf("failed to unmarshal mail info: %s\n", err)
		return
	}
}

func sendgmail(sender string, receipients []string, subject, message string) {
	username := gmailAccount.Username
	password := gmailAccount.Password
	if username == "" || password == "" {
		fmt.Printf("Mail info not read\n")
		return
	}
	hostname := "smtp.gmail.com"
	port := 587
	auth := smtp.PlainAuth("", username, password, hostname)
	msg := "To: "
	for _, r := range receipients {
		msg += r + ", "
	}
	msg = fmt.Sprintf("%s\r\nSubject: %s\r\n\r\nFrom: %s\r\n\r\n%s\r\n",
		msg, subject, sender, message)
	err := smtp.SendMail(
		fmt.Sprintf("%s:%d", hostname, port),
		auth, sender, receipients, []byte(msg))
	if err != nil {
		fmt.Printf("failed to send message: %s\n", err)
	}
}

var rawMsgToMsgKeyMap = map[string]string{
	"hi":            "hi",
	"hello":         "hi",
	"bye":           "bye",
	"see you later": "bye",
}

func rawMsgToMessageKey(rawMessage string) string {
	key, ok := rawMsgToMsgKeyMap[rawMessage]
	if !ok {
		return "exception"
	}
	return key
}

func loadMsgToKey(filepath string) bool {
	c, err := ioutil.ReadFile(filepath)
	if err != nil {
		fmt.Printf("failed to read messages from file: %s\n", err)
		return false
	}
	if err := json.Unmarshal(c, &rawMsgToMsgKeyMap); err != nil {
		fmt.Printf("failed to unmarshal messages: %s\n", err)
		return false
	}
	return true
}

func saveMsgToKey(filepath string) {
	bytes, err := json.Marshal(rawMsgToMsgKeyMap)
	if err != nil {
		fmt.Printf("failed to marshal messages: %s\n", err)
		return
	}

	if err := ioutil.WriteFile(filepath, bytes, 0600); err != nil {
		fmt.Printf("failed to write messages: %s\n", err)
		return
	}
}

var varMessages = map[string][]string{
	"intro":     {"Hi, everyone."},
	"welcome":   {"Welcome, $peername"},
	"hi":        {"Hello, $peername.  How are you? :D"},
	"bye":       {"Good bye, $peername.  See you later ;)"},
	"exception": {"Sorry, $peername.  I cannot understand what you mean."},
}

func getVarMessage(key, peername string) string {
	candidates, ok := varMessages[key]
	if !ok {
		return "..."
	}
	format := candidates[rand.Intn(len(candidates))]
	return strings.Replace(format, "$peername", peername, -1)
}

func loadVarMessges(filepath string) bool {
	c, err := ioutil.ReadFile(filepath)
	if err != nil {
		fmt.Printf("failed to read messages from file: %s\n", err)
		return false
	}
	if err := json.Unmarshal(c, &varMessages); err != nil {
		fmt.Printf("failed to unmarshal messages: %s\n", err)
		return false
	}

	return true
}

func saveVarMessages(filepath string) {
	bytes, err := json.Marshal(varMessages)
	if err != nil {
		fmt.Printf("failed to marshal messages: %s\n", err)
		return
	}

	if err := ioutil.WriteFile(filepath, bytes, 0600); err != nil {
		fmt.Printf("failed to write messages: %s\n", err)
		return
	}
}

func (bot *Bot) answerTo(question, peername string) {
	bot.send_privmsg(getVarMessage(question, peername))
}

var executables = map[string]bool{
	"ls": false,
}

func loadExecutables(filepath string) bool {
	c, err := ioutil.ReadFile(filepath)
	if err != nil {
		fmt.Printf("failed to read executables from file: %s\n", err)
		return false
	}
	if err := json.Unmarshal(c, &executables); err != nil {
		fmt.Printf("failed to unmarshal executables: %s\n", err)
		return false
	}

	return true
}

func saveExecutables(filepath string) {
	bytes, err := json.Marshal(executables)
	if err != nil {
		fmt.Printf("failed to marshal executables: %s\n", err)
		return
	}

	if err := ioutil.WriteFile(filepath, bytes, 0600); err != nil {
		fmt.Printf("failed to write executables: %s\n", err)
		return
	}
}

// Return true if handled, false if not
func (bot *Bot) handleCommand(msg, peername string) bool {
	tokens := strings.Fields(msg)
	if len(tokens) < 1 {
		return false
	}
	switch tokens[0] {
	case "answer":
		if len(tokens) < 4 {
			bot.send_privmsg(
				"Need one question and two or more selections")
			return true
		}
		bot.send_privmsg("%s? %s",
			tokens[1], tokens[2:][rand.Intn(len(tokens[2:]))])
	case "order":
		items := tokens[1:]
		if len(items) < 2 {
			bot.send_privmsg("You forgot items")
			return true
		}
		ordered := []string{}
		for len(items) > 0 {
			number := rand.Intn(len(items))
			ordered = append(ordered, items[number])
			items = append(items[:number], items[(number+1):]...)
		}
		bot.send_privmsg(strings.Join(ordered, " "))
	case "pick":
		selections := tokens[1:]
		if len(selections) < 2 {
			bot.send_privmsg("You forgot selections")
			return true
		}
		bot.send_privmsg("%s", selections[rand.Intn(len(selections))])
	case "htmltitle":
		if len(tokens) < 2 {
			bot.send_privmsg("You forgot html address.")
			return true
		}
		bot.send_privmsg("Title: %s", fetchHtmlTitle(tokens[1]))
	case "poll":
		bot.do_poll(peername, tokens)
	case "ex":
		if len(tokens) < 2 {
			bot.send_privmsg("You forgot command.")
			return true
		}
		allowed, ok := executables[tokens[1]]
		if !ok || !allowed {
			bot.send_privmsg("It cannot be executed.")
			return true
		}

		out, err := exec.Command("./"+tokens[1], tokens[2:]...).Output()
		if err != nil {
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
			return true
		}
		oper1, err := strconv.Atoi(tokens[1])
		if err != nil {
			bot.send_privmsg("operand 1 should be integer.")
			return true
		}
		oper2, err := strconv.Atoi(tokens[2])
		if err != nil {
			bot.send_privmsg("operand 2 should be integer.")
			return true
		}

		bot.send_privmsg("%s: Answer is %d", peername, oper1+oper2)
	case "commands":
		bot.send_privmsg("answer order pick htmltitle poll ex add")
	default:
		return false
	}
	return true
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
	if bot.handleCommand(msg, peername) {
		return
	}

	msg = rawMsgToMessageKey(msg)
	if bot.handleCommand(msg, peername) {
		return
	}

	// Human-like dialogue
	bot.answerTo(msg, peername)
}

func main() {
	if len(os.Args) < 6 {
		fmt.Printf("usage: s2j2 <server> <port> <pass> <channel> <nick>\n")
		os.Exit(1)
	}
	rand.Seed(time.Now().UTC().UnixNano())
	poll_results = map[int][]string{}
	read_gmailinfo()
	varmsgsFile := "var_msgs.json"
	if !loadVarMessges(varmsgsFile) {
		saveVarMessages(varmsgsFile)
	}

	msgtoKeyFile := "msg_to_key.json"
	if !loadMsgToKey(msgtoKeyFile) {
		saveMsgToKey(msgtoKeyFile)
	}

	executableFile := "executables.json"
	if !loadExecutables(executableFile) {
		saveExecutables(executableFile)
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
	bot.send_privmsg(getVarMessage("intro", ""))
	for {
		line, err := txtin.ReadLine()
		if err != nil {
			fmt.Printf("Error while reading input!\n")
			break
		}
		fmt.Printf("[%s] READ %s\n", time.Now(), line)
		if strings.HasPrefix(line, "PING ") {
			pongdata := strings.Split(line, "PING ")
			fmt.Printf("[%s] SEND PONG %s\n", time.Now(), pongdata[1])
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
				bot.answerTo("welcome", peername)
			}
		}
	}
}
