package peer

import(
    "net"
    "fmt"
    "time"
)

type Peer struct {
    ip net.IP
    port uint16
    connection net.Conn
    connected bool
    handshaked bool
}

func NewPeer(ip net.IP, port uint16) *Peer {
    p := Peer { }
    p.ip = ip
    p.port = port
    p.connected = false
    p.handshaked = false

    return &p
}

func (p *Peer) IsConnected() bool {
    return p.connected
}

func (p *Peer) Connect(done chan bool) {
    timeOut := time.Duration(1) * time.Second

    var err error
    p.connection, err = net.DialTimeout("tcp", p.ip.String() + ":" + fmt.Sprintf("%d", p.port), timeOut)
    if err != nil {
        done <- false
        return
    }

    p.connected = true
    done <- true
}

func (p *Peer) Handshake() {
    p.connection.Close()
}

