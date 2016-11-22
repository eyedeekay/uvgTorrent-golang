package peer

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/unovongalixor/bitfield-golang"
	"github.com/zeebo/bencode"
	"net"
	"time"
	"config"
	"strings"
	"../chunk"
	"../piece"
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
	metadata_chunks_received int64
	metadata 		   []byte
	bitfield           *bitfield.Bitfield

	chunk_chan		   chan *chunk.Chunk
	chunk 			   *chunk.Chunk

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
	p.metadata_chunks_received = 0
	p.metadata_requested = false
	p.bitfield = bitfield.NewBitfield(true, 1)
	p.isChoked = true
	p.chunk_chan = make(chan *chunk.Chunk)

	return &p
}

func (p *Peer) IsConnected() bool {
	return p.connected
}

func (p *Peer) IsChoked() bool {
	return p.isChoked
}

func (p *Peer) Connect() {
	timeOut := time.Duration(10) * time.Second

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
	p.connection.SetReadDeadline(time.Now().Add(5 * time.Second))
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
	metadata_piece_size := int64(config.ChunkSize)
	num_pieces := p.metadata_size / metadata_piece_size

	p.metadata = make([]byte, p.metadata_size)

	for i := int64(0); i <= num_pieces; i++ {
		bencoded_message := fmt.Sprintf("d8:msg_typei0e5:piecei%dee", i)

		var buff bytes.Buffer
		binary.Write(&buff, binary.BigEndian, int32(len(bencoded_message)+2))
		binary.Write(&buff, binary.BigEndian, int8(20))
		binary.Write(&buff, binary.BigEndian, int8(p.ut_metadata))
		binary.Write(&buff, binary.BigEndian, []byte(bencoded_message))
		p.connection.Write(buff.Bytes())
	}
}

func (p *Peer) Run(hash []byte, metadata chan []byte, request_chunk chan *Peer) {
	if p.IsConnected() == false {
		p.Connect()
		if p.IsConnected() {
			p.Handshake(hash)
		}
	}

	if p.IsConnected() && p.handshaked {
		sent_chunk_req := false
		if p.chunk != nil {
			sent_chunk_req = true
			p.SendChunkRequest()
		}
		p.HandleMessage(metadata, request_chunk)

		if sent_chunk_req == true && p.chunk != nil {
			if p.chunk.GetStatus() == chunk.ChunkStatusDone {
				p.chunk = nil
				p.GetChunkFromTorrent(request_chunk)
			} else {
				p.chunk.SetStatus(chunk.ChunkStatusReady)
				p.chunk = nil
			}
		}
	}

	if p.connected && p.handshaked {
		time.Sleep(5 * time.Millisecond) // sleep provides a small window for graceful shutdown
		go p.Run(hash, metadata, request_chunk)
	}
}

func (p *Peer) GetChunkFromTorrent(request_chunk chan *Peer) {
	request_chunk <- p
	p.chunk = <- p.chunk_chan
}

func (p *Peer) ClaimChunk(pieces []*piece.Piece) {
	if p.IsChoked() == false {
		for i, pi := range pieces {
			// if peer has piece

			if p.bitfield.GetBit(i) {
				ch := pi.GetNextChunk()

				if ch != nil {
					p.chunk_chan <- ch
					return 
				}
			}
		}
	}
}

func (p *Peer) SendChunkRequest() {
	chunk_size := int64(config.ChunkSize)
	msg_length := int32(13)
	msg_id := int8(6)

	index := int32(p.chunk.GetPieceIndex())
	begin := int32(chunk_size * p.chunk.GetIndex())
	piece_length := int32(p.chunk.GetLength())

	var buff bytes.Buffer
	binary.Write(&buff, binary.BigEndian, msg_length)
	binary.Write(&buff, binary.LittleEndian, msg_id)
	binary.Write(&buff, binary.BigEndian, index)
	binary.Write(&buff, binary.BigEndian, begin)
	binary.Write(&buff, binary.BigEndian, piece_length)
	_, err := p.connection.Write(buff.Bytes())
	if err != nil {
		fmt.Println("SEND FAILED")
	}
}


func (p *Peer) HandleMessage(metadata chan []byte, request_chunk chan *Peer) {
	var msg_length int32
	length_bytes := make([]byte, 4)
	length_bytes_read := 0
	p.connection.SetReadDeadline(time.Now().Add(3 * time.Second))

	for length_bytes_read < len(length_bytes) {
		n, err := p.connection.Read(length_bytes[length_bytes_read:4])
		if err != nil {
			p.Close()
			
			return
		}
		length_bytes_read += n
	}
	binary.Read(bytes.NewBuffer(length_bytes), binary.BigEndian, &msg_length)

	if msg_length > 0 && msg_length < int32(config.ChunkSize + 10000) {
		message := make([]byte, msg_length)
		message_bytes_read := 0
		for int32(message_bytes_read) < msg_length {
			n, err := p.connection.Read(message[message_bytes_read:msg_length])
			if err != nil {
				p.Close()
				
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
			fmt.Println(p.ip, "MSG_CHOKE")
			p.isChoked = true
		} else if msg_id == MSG_UNCHOKE {
			fmt.Println(p.ip, "MSG_UNCHOKE")
			p.isChoked = false
			p.GetChunkFromTorrent(request_chunk)
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
			var piece_index int32
			binary.Read(bytes.NewBuffer(message[1:]), binary.BigEndian, &piece_index)
			if len(message) > 9 {
				data := message[9:]
				if p.chunk != nil {
					p.chunk.SetData(data)
					p.chunk.SetStatus(chunk.ChunkStatusDone)	
				}
			}
		} else if msg_id == MSG_CANCEL {
			fmt.Println(p.ip, "MSG_CANCEL")
		} else if msg_id == MSG_PORT {
			fmt.Println(p.ip, "MSG_PORT")
		} else if msg_id == MSG_METADATA {
			fmt.Println(p.ip, "MSG_METADATA")
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

				if p.CanRequestMetadata() {
					p.RequestMetadata()
				}
			} else if handshake_id == 1 {
				var md map[string]interface{}
				message = message[2:]
				if err := bencode.DecodeBytes(message, &md); err != nil {
					panic(err)
				}

				metadata_piece_size := int64(config.ChunkSize)
				md_pos := strings.Index(string(message), "ee") + len("ee")
				copy(p.metadata[p.metadata_chunks_received:], message[md_pos:])

				metadata_pieces := p.metadata_size / metadata_piece_size + 1
				p.metadata_chunks_received++

				if p.metadata_chunks_received == metadata_pieces {
					metadata <- p.metadata
				}
			}
		} else {
			p.Close()
		}
	} else {
		p.Close()
	}
}

func (p *Peer) Close() {
	p.connected = false
	p.handshaked = false
	p.ut_metadata = 0 
	p.metadata_size = 0 
	p.metadata_requested = false
	p.connection.Close()
}
