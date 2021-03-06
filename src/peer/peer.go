package peer

import (
	"../chunk"
	"../piece"
	"bytes"
	"config"
	"encoding/binary"
	"fmt"
	"github.com/unovongalixor/bitfield-golang"
	"github.com/zeebo/bencode"
	"net"
	"io"
	"strings"
	"time"
)

type Peer struct {
	ip                       		net.IP
	port                     		uint16
	connection               		net.Conn
	connected                		bool
	closed                   		bool
	handshaked               		bool
	choked 				 			bool
	ut_metadata              		int64
	metadata_size            		int64
	metadata_chunks_received 		int64
	// have I sent a request for the torrents metadata to 
	// this peer yet?
	metadata_requested       		bool
	sent_chunk_req			 		bool
	metadata                 		[]byte
	// bitfield containing the pieces this peer has available for download
	bitfield                 		*bitfield.Bitfield


	// channel for receiving new chunks from the torrent object
	chunk_chan 				 		chan *chunk.Chunk
	// the chunk i'm currently working on
	chunk      				 		*chunk.Chunk
}

func NewPeer(ip net.IP, port uint16) *Peer {
	p := Peer{}
	p.ip = ip
	p.port = port
	p.choked = true
	p.bitfield = bitfield.NewBitfield(true, 1)
	p.chunk_chan = make(chan *chunk.Chunk, 1)

	return &p
}

func (p *Peer) IsConnected() bool {
	return p.connected
}

func (p *Peer) IsChoked() bool {
	return p.choked
}

func (p *Peer) IsMetadataLoaded() bool {
	metadata_piece_size := int64(config.ChunkSize)
	metadata_pieces := p.metadata_size/metadata_piece_size + 1

	return p.metadata_chunks_received == metadata_pieces
}

func (p *Peer) CanRequestMetadata() bool {
	if p.ut_metadata != 0 && p.metadata_size != 0 && p.metadata_requested == false {
		p.metadata_requested = true
		return true
	} else {
		return false
	}
}

// get chunk from torrent will send a request for the next
// available chunk belonging to a piece this peer has available
// for download
func (p *Peer) GetChunkFromTorrent(request_chunk chan *Peer) {
	if p.IsMetadataLoaded() && p.IsChoked() == false && p.connected && p.handshaked {
		// ask the torrent to call ClaimChunk at the next available opportunity
		request_chunk <- p
		select {
		case ch := <-p.chunk_chan:
			p.chunk = ch
			p.sent_chunk_req = false
		}
	}
}

// after calling GetChunkFromTorrent the torrent object
// calls this function, allowing the peer to select the 
// next available chunk in the main goroutine, unblocking
// the peer
func (p *Peer) ClaimChunk(pieces []*piece.Piece) {
	if p.IsChoked() == false && p.connected && p.handshaked {
		for i, pi := range pieces {
			// if peer has piece
			if pi.IsDownloadable() == true {
				if int64(i) > p.bitfield.Size() || p.bitfield.GetBit(i) {
					ch := pi.GetNextChunk()

					if ch != nil {
						p.chunk_chan <- ch
						return
					}
				}
			}
		}
	}
}

// establish a connection with the peer
func (p *Peer) Connect() {
	var err error
	p.connection, err = net.Dial("tcp", p.ip.String()+":"+fmt.Sprintf("%d", p.port))
	if err != nil {
		p.closed = true
		return
	}

	p.connected = true
}

// send extended handshake to peer
// see: http://www.rasterbar.com/products/libtorrent/extension_protocol.html
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
	p.connection.SetReadDeadline(time.Now().Add(60 * time.Second))
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

// tell the peer i'm looking for pieces
// see: https://wiki.theory.org/BitTorrentSpecification
func (p *Peer) SendInterested() {
	var msg_len int32 = 1
	var msg_id int8 = 2

	var buff bytes.Buffer
	binary.Write(&buff, binary.BigEndian, msg_len)
	binary.Write(&buff, binary.BigEndian, msg_id)

	p.connection.Write(buff.Bytes())
}

// request a chunk from a peer
// see: https://wiki.theory.org/BitTorrentSpecification
func (p *Peer) SendChunkRequest() {
	if p.sent_chunk_req == false {
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
		p.connection.Write(buff.Bytes())
		p.sent_chunk_req = true
	}
}

// send the extended metadata request
// see: http://www.rasterbar.com/products/libtorrent/extension_protocol.html 
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

// the peers main loop. each peer runs in a seperate goroutine.
// this function will connect and handshake with the peer if needed.
// if there is a chunk available to request it will request it and update the chunks 
// status accordingly.
// if the peer is still connected after attempting to handle a message from the peer
// the function will spin off a new goroutine to repeat the process
func (p *Peer) Run(hash []byte, metadata chan []byte, request_chunk chan *Peer) {
	if p.IsConnected() == false && p.closed == false {
		p.Connect()
		if p.IsConnected() {
			p.Handshake(hash)
		}
	}

	if p.IsConnected() && p.handshaked {
		if p.chunk != nil && p.sent_chunk_req == false {
			p.SendChunkRequest()
		}
		err, req_chunk := p.HandleMessage(metadata, request_chunk)

		if err == true {
			if p.chunk != nil {
				p.chunk.SetStatus(chunk.ChunkStatusReady)
			}
		}
		if req_chunk {
			p.GetChunkFromTorrent(request_chunk)
		}
	}

	if p.connected && p.handshaked {
		// sleep provides a small window for graceful shutdown
		// and to allow golang to switc hbetween goroutines
		// remove it and the program gets choppy
		time.Sleep(5 * time.Millisecond)
		go p.Run(hash, metadata, request_chunk)
	}
}

// attempt to handle a message from the peer
func (p *Peer) HandleMessage(metadata chan []byte, request_chunk chan *Peer) (bool, bool) {
	// timed_out := false
	var msg_length int32
	length_bytes := make([]byte, 4)
	length_bytes_read := 0
	p.connection.SetReadDeadline(time.Now().Add(120 * time.Second))

	for length_bytes_read < len(length_bytes) {
		n, err := p.connection.Read(length_bytes[length_bytes_read:4])
		if err != nil {
			if err == io.EOF {
				p.Close()
				return true, false
			}
			return true, false
		}
		length_bytes_read += n
	}
	binary.Read(bytes.NewBuffer(length_bytes), binary.BigEndian, &msg_length)

	if msg_length > 0 && msg_length < int32(config.ChunkSize+10000) {
		message := make([]byte, msg_length)
		message_bytes_read := 0

		for int32(message_bytes_read) < msg_length {
			n, err := p.connection.Read(message[message_bytes_read:msg_length])
			if err != nil {
				if err == io.EOF {
					p.Close()
					return true, false
				}
				return true, false
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
			p.choked = true
			return true, false
		} else if msg_id == MSG_UNCHOKE {
			p.choked = false
			return false, true
		} else if msg_id == MSG_INTERESTED {
		} else if msg_id == MSG_NOT_INTERESTED {
		} else if msg_id == MSG_HAVE {
			var have_bit int32
			binary.Read(bytes.NewBuffer(message[1:]), binary.BigEndian, &have_bit)

			p.bitfield.SetBit(int(have_bit))
		} else if msg_id == MSG_BITFIELD {
			p.bitfield.Copy(message[1:])
		} else if msg_id == MSG_REQUEST {
		} else if msg_id == MSG_PIECE {
			var piece_index int32
			binary.Read(bytes.NewBuffer(message[1:]), binary.BigEndian, &piece_index)
			if len(message) > 9 {
				data := message[9:]
				if p.chunk != nil {
					if len(data) == int(p.chunk.GetLength()) {
						p.chunk.SetData(data)
						p.chunk.SetStatus(chunk.ChunkStatusDone)
						p.chunk = nil
						return false, true
					}
				}
			}
			return true, true
		} else if msg_id == MSG_CANCEL {
		} else if msg_id == MSG_PORT {
		} else if msg_id == MSG_METADATA {
			var handshake_id int8
			binary.Read(bytes.NewBuffer(message[1:2]), binary.BigEndian, &handshake_id)

			if handshake_id == 0 {
				var torrent map[string]interface{}
				if err := bencode.DecodeBytes(message[2:], &torrent); err != nil {
					return true, false
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
				copy(p.metadata[p.metadata_chunks_received*metadata_piece_size:], message[md_pos:])

				p.metadata_chunks_received++

				if p.IsMetadataLoaded() {
					metadata <- p.metadata
					return false, true
				}
			}
			return false, false
		} else {
			//p.Close()
			return true, false
		}
	} else {
		//p.Close()
		return true, false
	}
	return false, false
}

func (p *Peer) Close() {
	p.connected = false
	p.handshaked = false
	p.ut_metadata = 0
	p.metadata_size = 0
	p.metadata_requested = false
	p.closed = true
	p.connection.Close()
}
