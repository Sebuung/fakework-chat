package main

import (
	"fakework-chat/internal/client"
	"log"
)

func main() {

	c := client.NewClient()
	if err := c.Start(); err != nil {
		log.Println(err)
	}

}
