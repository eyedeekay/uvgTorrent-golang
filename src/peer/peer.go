package peer

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/unovongalixor/bitfield-golang"
	"github.com/zeebo/bencode"
	"net"
	"time"
)

type Peer struct {
	ip                 net.IP
	port               uint16
	connection         net.Conn
	connected          bool
	handshaked         bool
	ut_metadata        int64
	metadata_size      int64
	metadata_requested bool
	bitfield           *bitfield.Bitfield

	isChoked bool
}

func NewPeer(ip net.IP, port uint16) *Peer {
	p := Peer{}
	p.ip = ip
	p.port = port
	p.connected = false
	p.handshaked = false
	p.ut_metadata = 0
	p.metadata_size = 0
	p.metadata_requested = false
	p.bitfield = bitfield.NewBitfield(true, 1)
	p.isChoked = true

	return &p
}

func (p *Peer) IsConnected() bool {
	return p.connected
}

func (p *Peer) IsChoked() bool {
	return p.isChoked
}

func (p *Peer) Connect() {
	timeOut := time.Duration(1) * time.Second

	var err error
	p.connection, err = net.DialTimeout("tcp", p.ip.String()+":"+fmt.Sprintf("%d", p.port), timeOut)
	if err != nil {
		return
	}

	p.connected = true
}

func (p *Peer) Handshake(hash []byte) {
	// send regular handshake
	var pstrlen int8
	pstrlen = 19
	pstr := "BitTorrent protocol"
	reserved := [8]byte{0, 0, 0, 0, 0, 16, 0, 0}
	peer_id := "UVG01234567891234567"

	var buff bytes.Buffer
	binary.Write(&buff, binary.BigEndian, pstrlen)
	binary.Write(&buff, binary.BigEndian, []byte(pstr))
	binary.Write(&buff, binary.BigEndian, reserved)
	binary.Write(&buff, binary.BigEndian, hash)
	binary.Write(&buff, binary.BigEndian, []byte(peer_id))

	p.connection.Write(buff.Bytes())

	result := make([]byte, 68)
	p.connection.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, err := p.connection.Read(result)
	if err != nil {
		return
	}

	p.handshaked = true

	// send extended handshake
	buff.Reset()
	metadata_message := "d1:md11:ut_metadatai1eee"
	binary.Write(&buff, binary.BigEndian, uint32(len(metadata_message)+2))
	binary.Write(&buff, binary.BigEndian, uint8(20))
	binary.Write(&buff, binary.BigEndian, uint8(0))
	binary.Write(&buff, binary.BigEndian, []byte(metadata_message))
	p.connection.Write(buff.Bytes())

	p.SendInterested()
}

func (p *Peer) SendInterested() {
	var msg_len int32 = 1
	var msg_id int8 = 2

	var buff bytes.Buffer
	binary.Write(&buff, binary.BigEndian, msg_len)
	binary.Write(&buff, binary.BigEndian, msg_id)

	p.connection.Write(buff.Bytes())
}

func (p *Peer) CanRequestMetadata() bool {
	if p.ut_metadata != 0 && p.metadata_size != 0 && p.metadata_requested == false {
		p.metadata_requested = true
		return true
	} else {
		return false
	}
}

func (p *Peer) RequestMetadata() {
	metadata_piece_size := int64(16 * 1024)
	num_pieces := p.metadata_size / metadata_piece_size

	// peer that requests metadata is thrown away
	// so discard any bitfield or has messages sent by the client
	// following the handshake
	result := make([]byte, 2048)
	p.connection.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, err := p.connection.Read(result)
	if err != nil {
		return
	}

	for i := int64(0); i <= num_pieces; i++ {
		bencoded_message := fmt.Sprintf("d8:msg_typei0e5:piecei%dee", i)

		var buff bytes.Buffer
		binary.Write(&buff, binary.BigEndian, int32(len(bencoded_message)+2))
		binary.Write(&buff, binary.BigEndian, int8(20))
		binary.Write(&buff, binary.BigEndian, int8(p.ut_metadata))
		binary.Write(&buff, binary.BigEndian, []byte(bencoded_message))
		p.connection.Write(buff.Bytes())
	}

	fmt.Println(p.ip, "REQUESTMETADATA")
}

func (p *Peer) Run(hash []byte, metadata chan []byte, request_chunk chan *Peer) {
	if p.IsConnected() == false {
		p.Connect()
		if p.IsConnected() {
			p.Handshake(hash)
			if p.CanRequestMetadata() {
				p.RequestMetadata()
			}
		}
	}


	if p.IsConnected() && p.handshaked {
		p.HandleMessage(metadata, request_chunk)
	}

	if p.connected {
		time.Sleep(50 * time.Millisecond)
		go p.Run(hash, metadata, request_chunk)
	}
}

func (p *Peer) HandleMessage(metadata chan []byte, request_chunk chan *Peer) {
	var msg_length int32
	length_bytes := make([]byte, 4)
	length_bytes_read := 0
	for length_bytes_read < len(length_bytes) {
		p.connection.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, err := p.connection.Read(length_bytes[length_bytes_read:4])
		if err != nil {
			p.connected = false
			return
		}
		length_bytes_read += n
	}
	binary.Read(bytes.NewBuffer(length_bytes), binary.BigEndian, &msg_length)

	if msg_length > 0 && msg_length < 16*1024 {
		message := make([]byte, msg_length)
		message_bytes_read := 0
		for int32(message_bytes_read) < msg_length {
			p.connection.SetReadDeadline(time.Now().Add(1 * time.Second))
			n, err := p.connection.Read(message[message_bytes_read:msg_length])
			if err != nil {
				return
			}
			message_bytes_read += n
		}

		const (
			MSG_CHOKE          = int8(0)
			MSG_UNCHOKE        = int8(1)
			MSG_INTERESTED     = int8(2)
			MSG_NOT_INTERESTED = int8(3)
			MSG_HAVE           = int8(4)
			MSG_BITFIELD       = int8(5)
			MSG_REQUEST        = int8(6)
			MSG_PIECE          = int8(7)
			MSG_CANCEL         = int8(8)
			MSG_PORT           = int8(9)
			MSG_METADATA       = int8(20)
		)

		var msg_id int8
		binary.Read(bytes.NewBuffer(message[0:1]), binary.BigEndian, &msg_id)

		if msg_id == MSG_CHOKE {
			p.isChoked = true
		} else if msg_id == MSG_UNCHOKE {
			p.isChoked = false

			request_chunk <- p
		} else if msg_id == MSG_INTERESTED {
			fmt.Println(p.ip, "MSG_INTERESTED")
		} else if msg_id == MSG_NOT_INTERESTED {
			fmt.Println(p.ip, "MSG_NOT_INTERESTED")
		} else if msg_id == MSG_HAVE {
			fmt.Println(p.ip, "MSG_HAVE")
			var have_bit int32
			binary.Read(bytes.NewBuffer(message[1:]), binary.BigEndian, &have_bit)

			p.bitfield.SetBit(int(have_bit))
		} else if msg_id == MSG_BITFIELD {
			fmt.Println(p.ip, "MSG_BITFIELD")
			p.bitfield.Copy(message[1:])
		} else if msg_id == MSG_REQUEST {
			fmt.Println(p.ip, "MSG_REQUEST")
		} else if msg_id == MSG_PIECE {
			fmt.Println(p.ip, "MSG_PIECE")
		} else if msg_id == MSG_CANCEL {
			fmt.Println(p.ip, "MSG_CANCEL")
		} else if msg_id == MSG_PORT {
			fmt.Println(p.ip, "MSG_PORT")
		} else if msg_id == MSG_METADATA {
			//fmt.Println(p.ip, "MSG_METADATA")
			var handshake_id int8
			binary.Read(bytes.NewBuffer(message[1:2]), binary.BigEndian, &handshake_id)

			if handshake_id == 0 {
				var torrent map[string]interface{}
				if err := bencode.DecodeBytes(message[2:], &torrent); err != nil {
					return
				}
				if torrent["metadata_size"] != nil {
					p.metadata_size = torrent["metadata_size"].(int64)
					m := torrent["m"].(map[string]interface{})
					p.ut_metadata = m["ut_metadata"].(int64)
				}
			} else if handshake_id == 1 {
				metadata <- message[2:]
			}
		}
	}
}

func (p *Peer) Close() {
	p.connection.Close()
}
