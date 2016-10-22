package tracker

import(
    "encoding/binary"
    "net/url"
    "net"
    "time"
    "bytes"
    "fmt"
    "../peer"
)

type Tracker struct {
    url string
    connected bool
    peers []*peer.Peer
    connection *net.UDPConn
    connection_id uint64
    interval uint32
    seeders uint32
    leechers uint32
}

func NewTracker(tracker_url string) *Tracker {
    t := Tracker { }
    t.connected = false
    
    u, err := url.Parse(tracker_url)
    if err != nil { panic(err) }

    t.url = u.Host
    if err != nil { panic(err) }

    return &t
}

func (t *Tracker) Connect(done chan bool) {
    sAddr, err := net.ResolveUDPAddr("udp", t.url)
    if err != nil { 
        done <- false
        return
    }

    cAddr, err := net.ResolveUDPAddr("udp", ":0")
    if err != nil { 
        done <- false
        return
    }

    t.connection, err = net.DialUDP("udp", cAddr, sAddr)
    if err != nil { 
        done <- false 
        return
    }
    t.connection.SetReadDeadline(time.Now().Add(1*time.Second))

    buf := make([]byte, 16)
    // connection id
    binary.BigEndian.PutUint64(buf[0:8], 0x41727101980)
    // action
    binary.BigEndian.PutUint32(buf[8:12], 0)
    // transaction id
    binary.BigEndian.PutUint32(buf[12:16], 123)

    n, err := t.connection.Write(buf[:])
    if err != nil || n < len(buf) {
        done <- false
        return
    }

    buf = make([]byte, 16)
    n, _, err = t.connection.ReadFromUDP(buf)
    if err != nil || n < len(buf) {
        done <- false
        return
    }

    action := binary.BigEndian.Uint32(buf[0:4])
    transaction_id := binary.BigEndian.Uint32(buf[4:8])
    t.connection_id = binary.BigEndian.Uint64(buf[8:16])

    if(action == 0 && transaction_id == 123){
        t.connected = true
    }

    done <- true
}

func (t *Tracker) IsConnected() bool {
    return t.connected
}

func (t *Tracker) Announce(hash []byte, done chan bool) {
    buf := make([]byte, 100)
    // connection id
    binary.BigEndian.PutUint64(buf[0:8], t.connection_id)
    // action
    binary.BigEndian.PutUint32(buf[8:12], 1)
    // transaction id
    binary.BigEndian.PutUint32(buf[12:16], 123)
    // info hash
    for i := 16; i < 36; i++ {
        buf[i] = hash[i-16]
    }

    peer_id := []byte("UVG01234567891234567")
    for i := 36; i < 56; i++ {
        buf[i] = peer_id[i-36]
    }
    // downloaded
    binary.BigEndian.PutUint64(buf[56:64], 0)
    // left
    binary.BigEndian.PutUint64(buf[64:72], 0)
    // uploaded
    binary.BigEndian.PutUint64(buf[72:80], 0)
    // event
    binary.BigEndian.PutUint32(buf[80:84], 2)
    // ip
    binary.BigEndian.PutUint32(buf[84:88], 0)
    // key
    binary.BigEndian.PutUint32(buf[88:92], 1)
    // num_want -1
    var num_want bytes.Buffer
    binary.Write(&num_want, binary.BigEndian, int32(-1))
    num_want_bytes := num_want.Bytes()
    buf[92] = num_want_bytes[0]
    buf[93] = num_want_bytes[1]
    buf[94] = num_want_bytes[2]
    buf[95] = num_want_bytes[3]
    // port
    binary.BigEndian.PutUint16(buf[96:98], 0)
    // extensions
    binary.BigEndian.PutUint16(buf[98:100], 0)

    n, err := t.connection.Write(buf[:])
    if err != nil || n < len(buf) {
        done <- false
        return
    }

    buf = make([]byte, 2048)
    t.connection.SetReadDeadline(time.Now().Add(1*time.Second))
    n, _, err = t.connection.ReadFromUDP(buf)
    if err != nil {
        done <- false
        return
    }

    t.connection.Close()
    if t.ParseAnnounceResponse(buf) == true {
        done <- true
    } else {
        done <- false
    }
}

func (t *Tracker) ParseAnnounceResponse(announce_response []byte) bool {
    action := binary.BigEndian.Uint32(announce_response[0:4])
    transaction_id := binary.BigEndian.Uint32(announce_response[4:8])

    if action == 1 && transaction_id == 123 {
        t.interval = binary.BigEndian.Uint32(announce_response[8:12])
        t.leechers = binary.BigEndian.Uint32(announce_response[12:16])
        t.seeders = binary.BigEndian.Uint32(announce_response[16:20])

        blank_ip := net.ParseIP("0.0.0.0")

        pos := 20
        for pos < 2048 {
            ip := make(net.IP, 4)
            binary.BigEndian.PutUint32(ip, binary.BigEndian.Uint32(announce_response[pos:pos + 4]))
            if ip.String() == blank_ip.String() {
                break
            }
            port := binary.BigEndian.Uint16(announce_response[pos+4:pos+6])

            t.peers = append(t.peers, peer.NewPeer(ip, port))

            pos += 6
        }

        return true
    } else {
        return false
    }
}

func (t *Tracker) Start() {
    connect_status := make(chan bool)

    for _, p := range t.peers {
        go p.Connect(connect_status)
    }

    for i := 0; i < len(t.peers); i++ {
        <-connect_status
    }

    for _, p := range t.peers {
        if p.IsConnected() {
            go p.Handshake()
        }
    }
    fmt.Println("STARTED")
}