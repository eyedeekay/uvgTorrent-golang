package piece

import (
    "../file"
    "../chunk"
    "config"
    "crypto/sha1"
)
type Piece struct {
	index           int64
	bytes_remaining int64
	length          int64
	boundaries      map[*file.File]*Boundary
	hash            []byte
    chunks          []*chunk.Chunk
    valid           bool
}

type Boundary struct {
	File_start  int64
	File_end    int64
	Piece_start int64
	Piece_end   int64
}

func NewPiece(index int64, length int64) *Piece {
	p := Piece{}
	p.index = index
	p.length = length
	p.bytes_remaining = p.length

	p.boundaries = make(map[*file.File]*Boundary)

	return &p
}

func (p *Piece) InitChunks(){
    p.length = p.length - p.bytes_remaining

    chunk_size := int64(config.ChunkSize)
    number_of_chunks := p.length / chunk_size
    last_chunk_size := p.length % chunk_size

    for c := int64(0); c < number_of_chunks; c++ {
        var ch *chunk.Chunk
        if c != number_of_chunks {
            ch = chunk.NewChunk(c, p.index, chunk_size)
        } else {
            ch = chunk.NewChunk(c, p.index, last_chunk_size)
        }

        p.AddChunk(ch)
    }
}

func (p *Piece) AddChunk(ch *chunk.Chunk){
    p.chunks = append(p.chunks, ch)
}

func (p *Piece) SetHash(hash []byte) {
    p.hash = hash
}

func (p *Piece) GetHash() []byte {
    return p.hash
}

func (p *Piece) GetRemainingBytes() int64 {
	return p.bytes_remaining
}

func (p *Piece) AddBoundary(f *file.File, bytes_remaining int64) int64 {
	b := &Boundary{}
	b.File_start = f.Length - bytes_remaining
	b.Piece_start = p.length - p.bytes_remaining

	if p.bytes_remaining > bytes_remaining {
		p.bytes_remaining -= bytes_remaining
		b.File_end = b.File_start + bytes_remaining
		bytes_remaining = 0
	} else {
		bytes_remaining -= p.bytes_remaining
		b.File_end = b.File_start + p.bytes_remaining
		p.bytes_remaining = 0
	}

	b.Piece_end = p.length - p.bytes_remaining

	p.boundaries[f] = b

	return bytes_remaining
}

func (p *Piece) GetNextChunk() *chunk.Chunk {
    for _, ch := range p.chunks {
        if ch.GetStatus() == chunk.ChunkStatusReady {
            ch.SetStatus(chunk.ChunkStatusInProgress)
            return ch
        }
    }

    return nil
}

func (p *Piece) ChunksCount() (int, int, bool) {
    total_chunks := len(p.chunks)
    completed_chunks := 0

    for _, ch := range p.chunks {
        if ch.GetStatus() == chunk.ChunkStatusDone {
            completed_chunks++
        }
    }

    success := false

    if completed_chunks == total_chunks {
        success = p.Verify()
    }

    return completed_chunks, total_chunks, success
}

func (p *Piece) Verify() bool {
    if p.valid == false {
        total_len := int64(0)

        for _, ch := range p.chunks {
            total_len += ch.GetLength()
        }

        data := make([]byte, total_len)
        index := 0
        for _, ch := range p.chunks {
            length := int(ch.GetLength())
            copy(data[index:index+length], ch.GetData())
            index += length
        }

        h := sha1.New()
        h.Write(data)
        hash := h.Sum(nil)

        if string(hash) == string(p.hash) {
            p.valid = true
            p.Write(data)
        } else {
            for _, ch := range p.chunks {
                ch.SetStatus(chunk.ChunkStatusReady)
            }
        }
    }

    return p.valid
}

func (p *Piece) Write(data []byte) {
    for f, b := range p.boundaries {
        f.Write(data[b.Piece_start:b.Piece_end], b.File_start)
    }
}
