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

func (p *Peer) Handshake(hash []byte, done chan bool) {
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
    p.connection.SetReadDeadline(time.Now().Add(1*time.Second))
    _, err := p.connection.Read(result)
    if err != nil {
        done <- false
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
    p.connection.SetReadDeadline(time.Now().Add(1*time.Second))
    _, err = p.connection.Read(result)
    if err != nil {
        done <- false
        return
    }
    binary.Read(bytes.NewBuffer(result), binary.BigEndian, &length)

    result = make([]byte, length)
    var bytes_read uint32 = 0
    for bytes_read < length {
        p.connection.SetReadDeadline(time.Now().Add(1*time.Second))
        n, err := p.connection.Read(result[bytes_read:length])
        if err != nil {
            done <- false
            return
        }
        bytes_read += uint32(n)
    }
    result_string := string(result[0:length])

    if strings.Index(result_string, "d1") != -1 {
        result_string = result_string[strings.Index(result_string, "d1"):]

        var torrent map[string]interface{}
        if err := bencode.DecodeString(result_string, &torrent); err != nil {
            done <- false
            return
        }

        p.metadata_size = torrent["metadata_size"].(int64)
        m := torrent["m"].(map[string]interface{})
        p.ut_metadata = m["ut_metadata"].(int64)
    }

    done <- true
}

func (p *Peer) CanRequestMetadata() bool {
    if p.ut_metadata != 0 && p.metadata_size != 0 {
        return true
    } else {
        return false
    }
}

func (p *Peer) RequestMetadata() {
    metadata_piece_size := int64(16*1024)
    num_pieces := p.metadata_size / metadata_piece_size

    // peer that requests metadata is thrown away
    // so discard any bitfield or has messages sent by the client 
    // following the handshake
    result := make([]byte, 2048)
    p.connection.SetReadDeadline(time.Now().Add(1*time.Second))
    _, err := p.connection.Read(result)
    if err != nil {
        return
    }

    for i := int64(0); i <= num_pieces; i++ {
        bencoded_message := fmt.Sprintf("d8:msg_typei0e5:piecei%dee", i)

        var buff bytes.Buffer
        binary.Write(&buff, binary.BigEndian, int32(len(bencoded_message) + 2))
        binary.Write(&buff, binary.BigEndian, int8(20))
        binary.Write(&buff, binary.BigEndian, int8(p.ut_metadata))
        binary.Write(&buff, binary.BigEndian, []byte(bencoded_message))
        p.connection.Write(buff.Bytes())
    }
}

func (p *Peer) HandleMessage(metadata chan string) {
    var msg_length int32
    length_bytes := make([]byte, 4)
    length_bytes_read := 0
    for length_bytes_read < len(length_bytes) {
        p.connection.SetReadDeadline(time.Now().Add(1*time.Second))
        n, err := p.connection.Read(length_bytes[length_bytes_read:4])
        if err != nil {
            return
        }
        length_bytes_read += n
    }
    binary.Read(bytes.NewBuffer(length_bytes), binary.BigEndian, &msg_length)

    message := make([]byte, msg_length)
    message_bytes_read := 0
    for int32(message_bytes_read) < msg_length {
        p.connection.SetReadDeadline(time.Now().Add(10*time.Second))
        n, err := p.connection.Read(message[message_bytes_read:msg_length])
        if err != nil {
            return
        }
        message_bytes_read += n
    }

    const (
        MSG_CHOKE = int8(0)
        MSG_UNCHOKE = int8(1)
        MSG_INTERESTED = int8(2)
        MSG_NOT_INTERESTED = int8(3)
        MSG_HAVE = int8(4)
        MSG_BITFIELD = int8(5)
        MSG_REQUEST = int8(6)
        MSG_PIECE = int8(7)
        MSG_CANCEL = int8(8)
        MSG_PORT = int8(9)
        MSG_METADATA = int8(20)
    )

    var msg_id int8
    binary.Read(bytes.NewBuffer(message[0:1]), binary.BigEndian, &msg_id)

    if msg_id == MSG_CHOKE {
        fmt.Println(p.ip, "MSG_CHOKE")
    } else if msg_id == MSG_UNCHOKE {
        fmt.Println(p.ip, "MSG_UNCHOKE")
    } else if msg_id == MSG_INTERESTED {
        fmt.Println(p.ip, "MSG_INTERESTED")
    } else if msg_id == MSG_NOT_INTERESTED {
        fmt.Println(p.ip, "MSG_NOT_INTERESTED")
    } else if msg_id == MSG_HAVE {
        fmt.Println(p.ip, "MSG_HAVE")
    } else if msg_id == MSG_BITFIELD {
        fmt.Println(p.ip, "MSG_BITFIELD")
    } else if msg_id == MSG_REQUEST {
        fmt.Println(p.ip, "MSG_REQUEST")
    } else if msg_id == MSG_PIECE {
        fmt.Println(p.ip, "MSG_PIECE")
    } else if msg_id == MSG_CANCEL {
        fmt.Println(p.ip, "MSG_CANCEL")
    } else if msg_id == MSG_PORT {
        fmt.Println(p.ip, "MSG_PORT")
    } else if msg_id == MSG_METADATA {
        fmt.Println(p.ip, "MSG_METADATA")
    }

    /*
    metadata := make([]byte, p.metadata_size)
    metadata_bytes_received := int64(0)

    for metadata_bytes_received < p.metadata_size {
        p.connection.SetReadDeadline(time.Now().Add(1*time.Second))
        n, err := p.connection.Read(metadata[metadata_bytes_received:p.metadata_size])
        metadata_bytes_received += int64(n)
        if err != nil {
            return
        }
    }
    fmt.Println(string(metadata))
    */
}

func (p *Peer) Close() {
    p.connection.Close()
}