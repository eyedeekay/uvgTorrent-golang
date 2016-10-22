package torrent

import(
    // "github.com/zeebo/bencode"
    "encoding/hex"
    "net/url"
    "strings"
    "../tracker"
)

type Torrent struct {
    Name string
    Hash []byte
    Trackers []tracker.Tracker
}

func NewTorrent(magnet_uri string) Torrent {
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

    return t
}
