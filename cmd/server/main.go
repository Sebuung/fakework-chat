package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

type Client struct {
	conn     net.Conn
	nick     string
	room     *Room
	outbound chan string
}

type Room struct {
	name    string
	clients map[*Client]bool
	mu      sync.RWMutex
}

func (r *Room) add(c *Client) {
	r.mu.Lock()
	r.clients[c] = true
	r.mu.Unlock()
	r.broadcast(fmt.Sprintf("*** %s joined", c.nick))
}

func (r *Room) remove(c *Client) {
	r.mu.Lock()
	if _, ok := r.clients[c]; ok {
		delete(r.clients, c)
	}
	r.mu.Unlock()
	r.broadcast(fmt.Sprintf("*** %s left", c.nick))
}

func (r *Room) broadcast(msg string) {
	r.mu.RLock()
	for c := range r.clients {
		select {
		case c.outbound <- msg:
		default:
		}
	}
	r.mu.RUnlock()
}

type Hub struct {
	mu    sync.RWMutex
	rooms map[string]*Room
}

func newHub() *Hub {
	return &Hub{rooms: make(map[string]*Room)}
}

func (h *Hub) getOrCreateRoom(name string) *Room {
	h.mu.Lock()
	defer h.mu.Unlock()
	if r, ok := h.rooms[name]; ok {
		return r
	}
	r := &Room{
		name:    name,
		clients: make(map[*Client]bool),
	}
	h.rooms[name] = r
	return r
}

func main() {
	addr := flag.String("addr", ":9000", "listen address, e.g. :9000 or 0.0.0.0:9000")
	flag.Parse()

	hub := newHub()
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("listen error: %v", err)
	}
	log.Printf("chat server listening on %s", *addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			continue
		}
		go handleConn(hub, conn)
	}
}

func handleConn(hub *Hub, conn net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("panic recovered: %v", r)
		}
	}()

	reader := bufio.NewScanner(conn)
	reader.Buffer(make([]byte, 0, 4096), 1024*64)

	if !reader.Scan() {
		conn.Close()
		return
	}
	first := strings.TrimSpace(reader.Text())
	parts := strings.Fields(first)
	if len(parts) < 3 || strings.ToUpper(parts[0]) != "JOIN" {
		fmt.Fprintln(conn, "ERR expected: JOIN <room> <nick>")
		conn.Close()
		return
	}
	roomName := parts[1]
	nick := strings.Join(parts[2:], " ")

	room := hub.getOrCreateRoom(roomName)
	client := &Client{
		conn:     conn,
		nick:     nick,
		room:     room,
		outbound: make(chan string, 64),
	}

	go func() {
		w := bufio.NewWriter(conn)
		for msg := range client.outbound {
			_, _ = w.WriteString(msg + "\n")
			if err := w.Flush(); err != nil {
				break
			}
		}
	}()

	room.add(client)
	fmt.Fprintf(conn, "*** joined room %s as %s\n", room.name, client.nick)

	for reader.Scan() {
		line := strings.TrimRight(reader.Text(), "\r\n")
		if len(line) == 0 {
			continue
		}
		if strings.HasPrefix(line, "/quit") {
			break
		}
		ts := time.Now().Format("15:04:05")
		room.broadcast(fmt.Sprintf("[%s] %s: %s", ts, client.nick, line))
	}

	room.remove(client)
	close(client.outbound)
	_ = conn.Close()
}
