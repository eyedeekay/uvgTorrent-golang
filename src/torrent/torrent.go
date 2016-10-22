package torrent

import(
    // "github.com/zeebo/bencode"
    "encoding/hex"
    "net/url"
    "strings"
    "../tracker"
    "fmt"
)

type Torrent struct {
    Name string
    Hash []byte
    Trackers []*tracker.Tracker
    connected_trackers int
}

func NewTorrent(magnet_uri string) *Torrent {
    t := Torrent{}
    
    u, err := url.Parse(magnet_uri)
    if err != nil { panic(err) }

    query, err := url.ParseQuery(u.RawQuery)
    if err != nil { panic(err) }

    t.Name = query["dn"][0]

    xt := strings.Split(query["xt"][0], ":")
    hash, err := hex.DecodeString(xt[len(xt) - 1])
    if err != nil { panic(err) }
    t.Hash = hash

    tr := query["tr"]

    for _, element := range tr {
        t.Trackers = append(t.Trackers, tracker.NewTracker(element))
    }

    return &t
}

func (t *Torrent) ConnectTrackers() {
    connect_status := make(chan bool)

    for _, track := range t.Trackers {
         go track.Connect(connect_status)
    }

    for i := 0; i < len(t.Trackers); i++ {
        <-connect_status
    }

    t.connected_trackers = 0
    for _, track := range t.Trackers {
        if track.IsConnected() {
            t.connected_trackers += 1
        }
    }

    fmt.Println("connect finished :: ", t.connected_trackers);
}

func (t *Torrent) AnnounceTrackers() {
    announce_status := make(chan bool)

    for _, track := range t.Trackers {
        if track.IsConnected() {
            go track.Announce(t.Hash, announce_status)
        }
    }
    for i := 0; i < t.connected_trackers; i++ {
        <-announce_status
    }


    fmt.Println("announce finished");
}

func (t *Torrent) Start() {
    for _, track := range t.Trackers {
        if track.IsConnected() {
            track.Start()
        }
    }
}