package piece

import(
    "../file"
)

type Piece struct {
    index  int64
    bytes_remaining int64
    length int64
    boundaries map[int]string
}

func NewPiece(index int64, length int64) *Piece {
    p := Piece{}
    p.index = index
    p.length = length
    p.bytes_remaining = p.length
    
    return &p
}

func (p *Piece) GetRemainingBytes() int64 {
    return p.bytes_remaining
}

func (p *Piece) AddBoundary(f *file.File, bytes_remaining int64) int64 {
    if p.bytes_remaining > bytes_remaining {
        p.bytes_remaining -= bytes_remaining
        bytes_remaining = 0
    } else {
        bytes_remaining -= p.bytes_remaining
        p.bytes_remaining = 0
    }

    return bytes_remaining
}