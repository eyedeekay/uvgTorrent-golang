package main

import (
    "os"
    "fmt"
    "./src/torrent"
)

func main() {
    t := torrent.NewTorrent(os.Args[1])

    fmt.Println(t.Name)
    fmt.Println(t.Hash)
    fmt.Println(t.Trackers)

    t.ConnectTrackers()
    t.AnnounceTrackers()
}