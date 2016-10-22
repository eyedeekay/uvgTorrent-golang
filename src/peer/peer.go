package peer

import(
    "net"
)

type Peer struct {
    ip net.IP
    port uint16
}

func NewPeer(ip net.IP, port uint16) *Peer {
    p := Peer { }
    p.ip = ip
    p.port = port

    return &p
}