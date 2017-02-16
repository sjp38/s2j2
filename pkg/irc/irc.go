package irc

import (
	"bufio"
	"fmt"
	"net"
	"net/textproto"
	"strings"
	"time"
)

type Conn struct {
	server  string
	port    string
	pass    string
	channel string
	nick    string
	conn    net.Conn
	reader  *textproto.Reader
}

func NewIRC(srv, port, pass, chann, nick string) *Conn {
	return &Conn{srv, port, pass, chann, nick, nil, nil}
}

func (conn *Conn) Connect() {
	var err error
	conn.conn, err = net.Dial("tcp", conn.server+":"+conn.port)
	if err != nil {
		fmt.Printf("Error while dialing to %s.\n",
			conn.server+":"+conn.port)
		time.Sleep(10 * time.Second)
		conn.Connect()
	}
	fmt.Printf("Connected to %s\n", conn.server+":"+conn.port)

	fmt.Fprintf(conn.conn, "USER %s 8 * :%s\r\n", conn.nick, conn.nick)
	fmt.Fprintf(conn.conn, "PASS %s\r\n", conn.pass)
	fmt.Fprintf(conn.conn, "NICK %s\r\n", conn.nick)
	fmt.Fprintf(conn.conn, "JOIN %s\r\n", conn.channel)
	rbuf := bufio.NewReader(conn.conn)
	conn.reader = textproto.NewReader(rbuf)
}

func (conn *Conn) Close() {
	conn.conn.Close()
}

func (conn *Conn) ReadLine() (string, error) {
	return conn.reader.ReadLine()
}

func (conn *Conn) SendPrivMSG(format string, args ...interface{}) {
	privmsg_pref := "PRIVMSG " + conn.channel + " :"
	msg := fmt.Sprintf(format, args...)
	for _, line := range strings.Split(msg, "\n") {
		if line == "" {
			line = " "
		}
		fmt.Printf("[%s] SEND %s\n", time.Now(), privmsg_pref+line)
		fmt.Fprintf(conn.conn, privmsg_pref+line+"\r\n")
	}
}
