package server

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

type CommandHandler func(c *Client, args []string)

type Client struct {
	conn     net.Conn
	nick     string
	room     *Room
	outbound chan string
	server   *Server
}

type Server struct {
	mu       sync.RWMutex
	rooms    map[string]*Room
	commands map[string]CommandHandler
}

func NewServer() *Server {
	s := &Server{
		rooms:    make(map[string]*Room),
		commands: make(map[string]CommandHandler),
	}
	s.registerCommands()
	return s
}

func (s *Server) registerCommands() {
	s.commands["/list"] = s.listCmd
	s.commands["/quit"] = s.quitCmd
	s.commands["/help"] = s.helpCmd
}

func (s *Server) Run(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer ln.Close()
	log.Printf("chat server listening on %s", addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			continue
		}
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	log.Printf("New client connected: %s", conn.RemoteAddr())

	defer func() {
		if r := recover(); r != nil {
			log.Printf("panic recovered in handleConn: %v", r)
		}
		conn.Close()
		log.Printf("Connection closed for: %s", conn.RemoteAddr())
	}()

	client, err := s.newClient(conn)
	if err != nil {
		fmt.Fprintf(conn, "ERR: %v\n", err)
		return
	}

	client.room.add(client)
	defer client.room.remove(client)
	go client.writeLoop()
	client.readLoop()
}

func (s *Server) newClient(conn net.Conn) (*Client, error) {
	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}

	fmt.Fprint(conn, "Welcome! Please enter JOIN <room> <nick>\n")

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("failed to read from client: %w", err)
		}
		return nil, fmt.Errorf("client disconnected before joining")
	}

	if err := conn.SetReadDeadline(time.Time{}); err != nil {
		return nil, fmt.Errorf("failed to clear read deadline: %w", err)
	}

	parts := strings.Fields(strings.TrimSpace(scanner.Text()))
	if len(parts) < 3 || strings.ToUpper(parts[0]) != "JOIN" {
		return nil, fmt.Errorf("expected: JOIN <room> <nick>")
	}
	roomName := parts[1]
	nick := strings.Join(parts[2:], " ")

	room := s.getOrCreateRoom(roomName)

	client := &Client{
		conn:     conn,
		nick:     nick,
		room:     room,
		outbound: make(chan string, 64),
		server:   s,
	}

	fmt.Fprintf(conn, "*** Joined room '%s' as '%s'\n", room.name, client.nick)
	s.helpCmd(client, nil)

	return client, nil
}

func (c *Client) readLoop() {
	scanner := bufio.NewScanner(c.conn)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 {
			continue
		}

		if strings.HasPrefix(line, "/") {
			parts := strings.Fields(line)
			cmdName := strings.ToLower(parts[0])

			if handler, ok := c.server.commands[cmdName]; ok {
				args := parts[1:]
				handler(c, args)
			} else {
				c.outbound <- fmt.Sprintf("ERR: Unknown command '%s'", cmdName)
			}
		} else {
			ts := time.Now().Format("15:04:05")
			c.room.broadcast(fmt.Sprintf("[%s] %s: %s", ts, c.nick, line))
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("read error from %s: %v", c.conn.RemoteAddr(), err)
	}
}

func (c *Client) writeLoop() {
	writer := bufio.NewWriter(c.conn)
	for msg := range c.outbound {
		_, err := writer.WriteString(msg + "\n")
		if err != nil {
			break
		}
		if err := writer.Flush(); err != nil {
			break
		}
	}
	close(c.outbound)
}

func (s *Server) getOrCreateRoom(name string) *Room {
	s.mu.Lock()
	defer s.mu.Unlock()

	if r, ok := s.rooms[name]; ok {
		return r
	}

	r := &Room{
		name:    name,
		clients: make(map[*Client]bool),
	}
	s.rooms[name] = r
	return r
}

func (s *Server) helpCmd(c *Client, args []string) {
	var helpMsg strings.Builder
	helpMsg.WriteString("*** Available commands:\n")
	helpMsg.WriteString("    /list - Show a list of users in the current room.\n")
	helpMsg.WriteString("    /quit - Disconnect from the chat server.\n")
	helpMsg.WriteString("    /help - Show this help message.\n")
	c.outbound <- helpMsg.String()
}

func (s *Server) listCmd(c *Client, args []string) {
	c.room.mu.RLock()
	defer c.room.mu.RUnlock()

	var users []string
	for client := range c.room.clients {
		users = append(users, client.nick)
	}
	c.outbound <- fmt.Sprintf("*** Users in %s: %s", c.room.name, strings.Join(users, ", "))
}

func (s *Server) quitCmd(c *Client, args []string) {
	log.Printf("Client %s (%s) is quitting.", c.nick, c.conn.RemoteAddr())
	c.conn.Close()
}
