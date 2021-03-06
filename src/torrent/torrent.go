package torrent

import (
	"../file"
	"../peer"
	"../piece"
	"../tracker"
	"../ui"

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
	pieces_length 	   int64
	total_length  	   int64
	
	files         	   []*file.File
	pieces        	   []*piece.Piece

	ui 				   *ui.UI
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
}

func (t *Torrent) Run() {
	// chan for delivering metadata to the torrent object
	metadata := make(chan []byte, 500)
	// chan for requesting the next available chunk of the torrent for a given peer to request
	request_chunk := make(chan *peer.Peer)

	for _, track := range t.Trackers {
		if track.IsConnected() {
			go track.Run(t.Hash, metadata, request_chunk)
		}
	}

	for {
		select {
			// torrent got metadata from a peer
			case data := <-metadata:
				if t.metadata == nil {
					t.ParseMetadata(data)
				}

			// a peer alerts the torrent it is ready to request a chunk
			case p := <-request_chunk:
				// allow the peer to lay claim to an available chunk
				p.ClaimChunk(t.pieces)

				// update ui percent bar
				if len(t.pieces) > 0 {
					completed_chunks := 0
					total_chunks := 0
					for _, p := range t.pieces {
						if p.IsDownloadable() {
							completed, total, _ := p.ChunksCount()
							total_chunks += total
							completed_chunks += completed
						}
						
					}

					t.ui.SetPercent(completed_chunks, total_chunks)
				}

		}
	}
}

func (t *Torrent) ParseMetadata(data []byte) {
	if err := bencode.DecodeBytes(data, &t.metadata); err != nil {
		t.metadata = nil
		return
	}
	t.pieces_length = t.metadata["piece length"].(int64)
	if _, ok := t.metadata["files"]; ok {
		for _, f := range t.metadata["files"].([]interface{}) {
			m := f.(map[string]interface{})

			length := m["length"].(int64)
			p := m["path"].([]interface{})

			path := make([]string, 0)
			path = append(path, "downloads")
			path = append(path, t.Name)
			for _, path_seq := range p {
				var str string = fmt.Sprintf("%v", path_seq)
				path = append(path, str)
			}

			t.addFile(file.NewFile(length, path))
		}
	} else {
		// single file torrent
		length := t.metadata["length"].(int64)
		name := t.metadata["name"].(string)

		path := make([]string, 0)
		path = append(path, "downloads")
		path = append(path, t.Name)
		path = append(path, name)

		t.addFile(file.NewFile(length, path))

	}

	t.initPieces([]byte(t.metadata["pieces"].(string)))

	t.SelectFile()
}

func (t *Torrent) SelectFile() {
	file_chan := make(chan int)
	t.ui.SelectFile(t.files, file_chan)

	file_index := <- file_chan
	
	if file_index < len(t.files) {
		f := t.files[file_index]
		f.SetDownloadable(true)

		start_piece, end_piece := f.GetStartAndEndPieces()
		for i := start_piece; i <= end_piece; i++ {
			t.pieces[i].SetDownloadable(true)
		}
	} else {
		for _, f := range t.files { 
			f.SetDownloadable(true)
		}
		for _, pi := range t.pieces { 
			pi.SetDownloadable(true)
		}
	}
}

func (t *Torrent) SetUI(u *ui.UI) {
	t.ui = u
}

func (t *Torrent) Close() {
	for _, track := range t.Trackers {
		if track.IsConnected() {
			track.Close()
		}
	}

	for _, f := range t.files {
		f.Close()
	}
}

func (t *Torrent) addFile(f *file.File) {
	t.files = append(t.files, f)
	t.total_length += f.GetLength()
}

func (t *Torrent) addPiece(p *piece.Piece) {
	p.InitChunks()
	t.pieces = append(t.pieces, p)
}

func (t *Torrent) initPieces(pieces []byte) {
	var current_piece *piece.Piece = nil
	current_piece_index := int64(-1)
	for _, f := range t.files {
		file_bytes_remaining := f.GetLength()
		if current_piece == nil {
			f.SetStartPiece(0)
		} else {
			f.SetStartPiece(current_piece_index)
		}
		

		for file_bytes_remaining > 0 {
			if current_piece == nil {
				current_piece_index++
				current_piece = piece.NewPiece(current_piece_index, t.pieces_length)
				current_piece.SetHash([]byte(pieces[current_piece_index*20 : current_piece_index*20+20]))
			}

			file_bytes_remaining = current_piece.AddBoundary(f, file_bytes_remaining)

			if current_piece.GetRemainingBytes() == 0 {
				t.addPiece(current_piece)
				current_piece = nil
			}
		}

		f.SetEndPiece(current_piece_index)
	}

	t.addPiece(current_piece)
	current_piece = nil
}
