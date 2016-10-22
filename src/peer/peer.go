package peer

import(
    "net"
    "fmt"
    "time"
    "bytes"
    "encoding/binary"
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

func (p *Peer) Handshake(hash []byte) {
    var pstrlen int8
    pstrlen = 19
    pstr := "BitTorrent protocol"
    reserved := [8]byte { 0, 0, 0, 0, 0, 1, 0, 0 }
    peer_id := "UVG01234567891234567"

    var buff bytes.Buffer
    binary.Write(&buff, binary.LittleEndian, pstrlen)
    binary.Write(&buff, binary.LittleEndian, []byte(pstr))
    binary.Write(&buff, binary.LittleEndian, reserved)
    binary.Write(&buff, binary.LittleEndian, hash)
    binary.Write(&buff, binary.LittleEndian, []byte(peer_id))

    fmt.Println(len(buff.Bytes()))
    p.connection.Close()
}

