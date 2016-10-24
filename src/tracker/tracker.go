package tracker

import(
    "encoding/binary"
    "net/url"
    "net"
    "time"
    "bytes"
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

    var buf bytes.Buffer
    binary.Write(&buf, binary.BigEndian, uint64(0x41727101980))
    binary.Write(&buf, binary.BigEndian, uint32(0))
    binary.Write(&buf, binary.BigEndian, uint32(123))

    n, err := t.connection.Write(buf.Bytes())
    if err != nil || n < len(buf.Bytes()) {
        done <- false
        return
    }

    result := make([]byte, 16)
    n, _, err = t.connection.ReadFromUDP(result)
    if err != nil || n < len(result) {
        done <- false
        return
    }

    action := binary.BigEndian.Uint32(result[0:4])
    transaction_id := binary.BigEndian.Uint32(result[4:8])
    t.connection_id = binary.BigEndian.Uint64(result[8:16])

    if(action == 0 && transaction_id == 123){
        t.connected = true
    }

    done <- true
}

func (t *Tracker) IsConnected() bool {
    return t.connected
}

func (t *Tracker) Announce(hash []byte, done chan bool) {
    var buf bytes.Buffer
    // connection id
    binary.Write(&buf, binary.BigEndian, uint64(t.connection_id))
    // action
    binary.Write(&buf, binary.BigEndian, uint32(1))
    // transaction id
    binary.Write(&buf, binary.BigEndian, uint32(123))
    // info hash
    binary.Write(&buf, binary.LittleEndian, hash)
    // peer id
    binary.Write(&buf, binary.LittleEndian, []byte("UVG01234567891234567"))
    // downloaded
    binary.Write(&buf, binary.BigEndian, uint64(0))
    // left
    binary.Write(&buf, binary.BigEndian, uint64(0))
    // uploaded
    binary.Write(&buf, binary.BigEndian, uint64(0))
    // event
    binary.Write(&buf, binary.BigEndian, uint32(2))
    // ip
    binary.Write(&buf, binary.BigEndian, uint32(0))
    // key
    binary.Write(&buf, binary.BigEndian, uint32(1))
    // num_want -1
    binary.Write(&buf, binary.BigEndian, int32(-1))
    // port
    binary.Write(&buf, binary.BigEndian, uint16(0))
    // extensions
    binary.Write(&buf, binary.BigEndian, uint16(0))

    n, err := t.connection.Write(buf.Bytes())
    if err != nil || n < len(buf.Bytes()) {
        done <- false
        return
    }

    result := make([]byte, 2048)
    t.connection.SetReadDeadline(time.Now().Add(1*time.Second))
    n, _, err = t.connection.ReadFromUDP(result)
    if err != nil {
        done <- false
        return
    }

    t.connection.Close()
    if t.ParseAnnounceResponse(result) == true {
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

func (t *Tracker) Start(hash []byte, done chan bool) {
    connect_status := make(chan bool)
    for _, p := range t.peers {
        go p.Connect(connect_status)
    }

    for i := 0; i < len(t.peers); i++ {
        <-connect_status
    }

    handshake_status := make(chan bool)
    connected_peer_count := 0
    for _, p := range t.peers {
        if p.IsConnected() {
            connected_peer_count++
            go p.Handshake(hash, handshake_status)
        }
    }

    for i := 0; i < connected_peer_count; i++ {
        <-handshake_status
    }

    for _, p := range t.peers {
        if p.IsConnected() {
            if p.CanRequestMetadata() {
                go p.RequestMetadata()
            }
        }
    }

    done <- true
}

func (t *Tracker) Run(metadata chan string) {
    for _, p := range t.peers {
        if p.IsConnected() {
            go p.HandleMessage(metadata)
        }
    }
}

func (t *Tracker) Close() {
    for _, p := range t.peers {
        if p.IsConnected() {
            p.Close()
        }
    }
}