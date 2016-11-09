package torrent

import (
	"../tracker"
	"../file"
	"../piece"
	"../peer"
	"encoding/hex"
	"fmt"
	"github.com/zeebo/bencode"
	"net/url"
	"strings"
)

type Torrent struct {
	Name               string
	Hash               []byte
	Trackers           []*tracker.Tracker
	connected_trackers int
	metadata           map[string]interface{}

	pieces_count	   int64
	pieces_length	   int64
	files			   []*file.File
	pieces			   []*piece.Piece
	total_length	   int64
}

func NewTorrent(magnet_uri string) *Torrent {
	t := Torrent{}

	u, err := url.Parse(magnet_uri)
	if err != nil {
		panic(err)
	}

	query, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		panic(err)
	}

	t.Name = query["dn"][0]

	xt := strings.Split(query["xt"][0], ":")
	hash, err := hex.DecodeString(xt[len(xt)-1])
	if err != nil {
		panic(err)
	}
	t.Hash = hash

	tr := query["tr"]

	for _, element := range tr {
		t.Trackers = append(t.Trackers, tracker.NewTracker(element))
	}

	t.metadata = nil
	t.total_length = 0

	return &t
}

func (t *Torrent) ConnectTrackers() {
	connect_status := make(chan bool)

	for _, track := range t.Trackers {
		go track.Connect(connect_status)
	}

	for i := 0; i < len(t.Trackers); i++ {
		<-connect_status
	}

	t.connected_trackers = 0
	for _, track := range t.Trackers {
		if track.IsConnected() {
			t.connected_trackers += 1
		}
	}

	fmt.Println("connect finished :: ", t.connected_trackers)
}

func (t *Torrent) AnnounceTrackers() {
	announce_status := make(chan bool)

	for _, track := range t.Trackers {
		if track.IsConnected() {
			go track.Announce(t.Hash, announce_status)
		}
	}
	for i := 0; i < t.connected_trackers; i++ {
		<-announce_status
	}

	fmt.Println("announce finished")
}

func (t *Torrent) Start() {
	tracker_status := make(chan bool)

	for _, track := range t.Trackers {
		if track.IsConnected() {
			go track.Start(t.Hash, tracker_status)
		}
	}

	for i := 0; i < t.connected_trackers; i++ {
		<-tracker_status
	}
}

func (t *Torrent) Run() {
	metadata := make(chan []byte)
	pieces := make(chan bool)
	request_chunk := make(chan *peer.Peer)

	for {
		for _, track := range t.Trackers {
			if track.IsConnected() {
				go track.Run(metadata, pieces, request_chunk)
			}
		}

		for _, _ = range t.Trackers {
			select {

            // torrent got metadata from a peer
			case data := <-metadata:
				if t.metadata == nil {
					dict_pos := strings.Index(string(data), "ee") + len("ee")
					if err := bencode.DecodeBytes(data[dict_pos:], &t.metadata); err != nil {
						panic(err)
					}
					t.pieces_length = t.metadata["piece length"].(int64)

					for _, f := range t.metadata["files"].([]interface{}) {
						m := f.(map[string]interface{})

						length := m["length"].(int64)
						p := m["path"].([]interface{})

						path := make([]string, len(p)-1)
						for _, path_seq := range p {
							var str string = fmt.Sprintf("%v", path_seq)
							path = append(path, str)
						}

						t.addFile(file.NewFile(length, path))
					}

					t.initPieces()
				}

            // torrent got a chunk from a peer
			case <-pieces:

			// a peer alerts the torrent it is ready to request a chunk
			case p := <-request_chunk:
				fmt.Println(p)
			}
		}
	}
}

func (t *Torrent) Close() {
	for _, track := range t.Trackers {
		if track.IsConnected() {
			track.Close()
		}
	}
}

func (t *Torrent) addFile(f *file.File) {
	t.files = append(t.files, f)
	t.total_length += f.Length
}

func (t *Torrent) addPiece(p *piece.Piece) {
	t.pieces = append(t.pieces, p)
}

func (t *Torrent) initPieces() {
	var current_piece *piece.Piece = nil
	current_piece_index := int64(0)
	for _, f := range t.files {
		file_bytes_remaining := f.Length
		f.Start_piece = current_piece_index

		for file_bytes_remaining > 0 {
			if current_piece == nil {
				current_piece = piece.NewPiece(current_piece_index, t.pieces_length)
				current_piece_index++
			}

			file_bytes_remaining = current_piece.AddBoundary(f, file_bytes_remaining)

			if current_piece.GetRemainingBytes() == 0 {
				t.addPiece(current_piece)
				current_piece = nil
			}
		}

		f.End_piece = current_piece_index
	}
}