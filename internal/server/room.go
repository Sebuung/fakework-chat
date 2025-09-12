package server

import (
	"fmt"
	"sync"
)

type Room struct {
	name    string
	clients map[*Client]bool
	mu      sync.RWMutex
}

func (r *Room) add(c *Client) {
	r.mu.Lock()
	r.clients[c] = true
	r.mu.Unlock()

	r.broadcast(fmt.Sprintf("*** %s joined the room.", c.nick))
}

func (r *Room) remove(c *Client) {
	r.mu.Lock()

	if _, ok := r.clients[c]; ok {
		delete(r.clients, c)
		r.mu.Unlock()
		r.broadcast(fmt.Sprintf("*** %s left the room.", c.nick))
	} else {
		r.mu.Unlock()
	}
}

func (r *Room) broadcast(msg string) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for c := range r.clients {

		select {
		case c.outbound <- msg:
		default:

		}
	}
}
