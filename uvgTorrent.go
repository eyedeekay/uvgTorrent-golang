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

    go run(t)
	
    ui := ui.NewUI()
    t.SetUI(ui)
    ui.Init(t.Name, t.Trackers)

    t.Close()
}

func run(t *torrent.Torrent) {
    t.ConnectTrackers()
    t.AnnounceTrackers()
    t.Run()
}

func cleanup(t *torrent.Torrent) {
	fmt.Println()
	fmt.Println("cleaning up")

	// torrent close will tell the trackers to close all of the peers connections
	// causing the peers to gracefully exit
	// it will also close any open file handles
	t.Close()
}
