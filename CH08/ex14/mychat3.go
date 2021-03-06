// Copyright © 2016 "Shun Yokota" All rights reserved

// Chat is a server that lets clients chat with each other.
package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"time"
)

//!+broadcaster
type client struct {
	ch   chan<- string // an outgoing message channel
	name string
}

var (
	entering = make(chan client)
	leaving  = make(chan client)
	messages = make(chan string) // all incoming client messages
)

func broadcaster() {
	clients := make(map[client]bool) // all connected clients
	for {
		select {
		case msg := <-messages:
			// Broadcast incoming message to all
			// clients' outgoing message channels.
			for cli := range clients {
				cli.ch <- msg
			}

		case cli := <-entering:
			cli.ch <- "menbers: "
			for c := range clients {
				cli.ch <- c.name
			}
			clients[cli] = true

		case cli := <-leaving:
			delete(clients, cli)
			close(cli.ch)
		}
	}
}

//!-broadcaster

//!+handleConn
func handleConn(conn net.Conn) {

	ch := make(chan string) // outgoing client messages
	go clientWriter(conn, ch)

	input := bufio.NewScanner(conn)
	ch <- "input your name:"
	who := conn.RemoteAddr().String()
	if input.Scan() {
		who = input.Text()
	}

	ch <- "You are " + who
	messages <- who + " has arrived"
	entering <- client{ch, who}

	tick := time.Tick(1 * time.Minute)
	send := make(chan struct{})
	closed := make(chan struct{})
	go func() {
		cd := 0
		for {
			select {
			case <-tick:
				cd++
				if cd >= 5 {
					conn.Close()
					closed <- struct{}{}
				}
			case <-send:
				cd = 0
			}
		}
	}()

	for input.Scan() {
		messages <- who + ": " + input.Text()
		send <- struct{}{}
	}
	// NOTE: ignoring potential errors from input.Err()

	leaving <- client{ch, who}
	messages <- who + " has left"
	select {
	case <-closed:
		return
	default:
		conn.Close()
	}
}

func clientWriter(conn net.Conn, ch <-chan string) {
	for msg := range ch {
		fmt.Fprintln(conn, msg) // NOTE: ignoring network errors
	}
}

//!-handleConn

//!+main
func main() {
	listener, err := net.Listen("tcp", "localhost:8000")
	if err != nil {
		log.Fatal(err)
	}

	go broadcaster()
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Print(err)
			continue
		}
		go handleConn(conn)
	}
}

//!-main
