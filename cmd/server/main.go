package main

import (
	"flag"
	"log"

	"fakework-chat/internal/server"
)

func main() {
	addr := flag.String("addr", ":9000", "listen address, e.g. :9000 or 0.0.0.0:9000")

	flag.Parse()

	server := server.NewServer()
	if err := server.Run(*addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
