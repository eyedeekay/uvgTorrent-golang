package peer

import(
    "net"
    "fmt"
    "time"
    "bytes"
    "strings"
    "encoding/binary"
    "github.com/zeebo/bencode"
)

type Peer struct {
    ip net.IP
    port uint16
    connection net.Conn
    connected bool
    handshaked bool
    ut_metadata int64
    metadata_size int64
}

func NewPeer(ip net.IP, port uint16) *Peer {
    p := Peer { }
    p.ip = ip
    p.port = port
    p.connected = false
    p.handshaked = false
    p.ut_metadata = 0
    p.metadata_size = 0

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
    defer p.connection.Close()

    // send regular handshake
    var pstrlen int8
    pstrlen = 19
    pstr := "BitTorrent protocol"
    reserved := [8]byte { 0, 0, 0, 0, 0, 16, 0, 0 }
    peer_id := "UVG01234567891234567"

    var buff bytes.Buffer
    binary.Write(&buff, binary.BigEndian, pstrlen)
    binary.Write(&buff, binary.BigEndian, []byte(pstr))
    binary.Write(&buff, binary.BigEndian, reserved)
    binary.Write(&buff, binary.BigEndian, hash)
    binary.Write(&buff, binary.BigEndian, []byte(peer_id))

    p.connection.Write(buff.Bytes())

    result := make([]byte, 68)
    p.connection.SetReadDeadline(time.Now().Add(3*time.Second))
    _, err := p.connection.Read(result)
    if err != nil {
        return
    }

    p.handshaked = true

    // send extended handshake
    buff.Reset()
    metadata_message := "d1:md11:ut_metadatai1eee"
    binary.Write(&buff, binary.BigEndian, uint32(len(metadata_message) + 2))
    binary.Write(&buff, binary.BigEndian, uint8(20))
    binary.Write(&buff, binary.BigEndian, uint8(0))
    binary.Write(&buff, binary.BigEndian, []byte(metadata_message))
    p.connection.Write(buff.Bytes())

    var length uint32
    result = make([]byte, 4)
    p.connection.SetReadDeadline(time.Now().Add(3*time.Second))
    _, err = p.connection.Read(result)
    if err != nil {
        return
    }
    binary.Read(bytes.NewBuffer(result), binary.BigEndian, &length)

    result = make([]byte, length)
    var bytes_read uint32 = 0
    for bytes_read < length {
        p.connection.SetReadDeadline(time.Now().Add(3*time.Second))
        n, err := p.connection.Read(result[bytes_read:length])
        if err != nil {
            return
        }
        bytes_read += uint32(n)
    }
    result_string := string(result[0:length])

    if strings.Index(result_string, "d1") != -1 {
        result_string = result_string[strings.Index(result_string, "d1"):]

        var torrent map[string]interface{}
        if err := bencode.DecodeString(result_string, &torrent); err != nil {
            return
        }

        p.metadata_size = torrent["metadata_size"].(int64)
        m := torrent["m"].(map[string]interface{})
        p.ut_metadata = m["ut_metadata"].(int64)
    }
}

func (p *Peer) CanRequestMetadata() bool {
    if p.ut_metadata != 0 && p.metadata_size != 0 {
        return true
    } else {
        return false
    }
}

func (p *Peer) RequestMetadata() {

}