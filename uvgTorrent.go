package main

import (
	"./src/torrent"
    "./src/ui"
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
		os.Exit(0)
	}()

	t.ConnectTrackers()
	t.AnnounceTrackers()
    
    ui := ui.NewUI()
    ui.Init(t.Name, t.Trackers)

	t.Run() // loop through peers forever handling messages
}

func cleanup(t *torrent.Torrent) {
	fmt.Println()
	fmt.Println("cleaning up")

	// torrent close will tell the trackers to close all of the peers connections
	// causing the peers to gracefully exit
	// it will also close any open file handles
	t.Close()
}
