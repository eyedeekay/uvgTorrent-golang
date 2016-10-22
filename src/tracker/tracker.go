package tracker

import(
    "fmt"
    "net/url"
    "net"
    "strconv"
)

type Tracker struct {
    url string
    port int
    peers []string
}

func NewTracker(tracker_url string) Tracker {
    t := Tracker { }
    
    u, err := url.Parse(tracker_url)
    if err != nil { panic(err) }

    host, port, _ := net.SplitHostPort(u.Host)

    t.url = fmt.Sprintf("%s://%s", u.Scheme, host)
    t.port, _ = strconv.Atoi(port)

    return t
}