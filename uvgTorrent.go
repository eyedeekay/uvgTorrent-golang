package main

import (
	"./src/torrent"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	t := torrent.NewTorrent(os.Args[1])

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cleanup(t)
		os.Exit(1)
	}()

	fmt.Println(t.Name)
	fmt.Println(t.Hash)
	fmt.Println(t.Trackers)

	t.ConnectTrackers()
	t.AnnounceTrackers()

	t.Run() // loop through peers forever handling messages
}

func cleanup(t *torrent.Torrent) {
	fmt.Println("cleaning up")
	t.Close()
}
