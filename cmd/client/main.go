package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/chzyer/readline"
)

func main() {
	room := flag.String("r", "", "room name to join (required)")
	nick := flag.String("n", "", "nickname to use (required)")
	server := flag.String("s", "chat.sebuung.com:9000", "chat server address (host:port)")
	flag.Parse()
	if *room == "" || *nick == "" {
		flag.Usage()
		os.Exit(2)
	}

	dialer := net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
	conn, err := dialer.Dial("tcp", *server)
	if err != nil {
		log.Fatalf("connect %s: %v", *server, err)
	}
	defer conn.Close()

	writer := bufio.NewWriter(conn)
	if _, err := fmt.Fprintf(writer, "JOIN %s %s\n", *room, *nick); err != nil {
		log.Fatalf("JOIN: %v", err)
	}
	if err := writer.Flush(); err != nil {
		log.Fatalf("JOIN flush: %v", err)
	}

	prompt := fmt.Sprintf("%s@%s> ", *nick, *room)
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          prompt,
		HistoryLimit:    200,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		log.Fatalf("readline init: %v", err)
	}
	defer rl.Close()

	printServerLine := func(s string) {
		_, _ = rl.Write([]byte("\r\x1b[2K" + s + "\n"))
		rl.Refresh()
	}

	go func() {
		sc := bufio.NewScanner(conn)
		buf := make([]byte, 0, 4096)
		sc.Buffer(buf, 1024*64)
		for sc.Scan() {
			line := sc.Text()
			printServerLine(line)
		}
		if err := sc.Err(); err != nil && err != io.EOF {
			printServerLine(fmt.Sprintf("[conn err] %v", err))
		}
	}()

	for {
		text, err := rl.Readline()
		if err == readline.ErrInterrupt || err == io.EOF {
			break
		}
		if err != nil {
			printServerLine(fmt.Sprintf("[rl err] %v", err))
			continue
		}

		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		if text == "/quit" {
			break
		}

		eraseLastInputLine(rl)

		if _, err := fmt.Fprintln(writer, text); err != nil {
			printServerLine(fmt.Sprintf("[write err] %v", err))
			break
		}
		if err := writer.Flush(); err != nil {
			printServerLine(fmt.Sprintf("[flush err] %v", err))
			break
		}
	}

	_, _ = fmt.Fprintln(writer, "/quit")
	_ = writer.Flush()
}

func eraseLastInputLine(rl *readline.Instance) {
	_, _ = rl.Write([]byte("\x1b[1A\r\x1b[2K"))
	rl.Refresh()
}
