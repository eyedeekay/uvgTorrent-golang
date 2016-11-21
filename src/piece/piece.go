package piece

import (
    "../file"
    "config"
    "fmt"
)

type Piece struct {
	index           int64
	bytes_remaining int64
	length          int64
	boundaries      map[*file.File]*Boundary
	hash            []byte
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

    fmt.Println(number_of_chunks, p.length, last_chunk_size)
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
