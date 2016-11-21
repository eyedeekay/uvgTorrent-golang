package chunk

import(
    "sync"
)

const(
    ChunkStatusReady = 0
    ChunkStatusInProgress = 1
    ChunkStatusDone = 2
)

type Chunk struct {
    index           int64
    piece_index     int64
    data            []byte
    status          int

    status_lock     *sync.Mutex
}


func NewChunk(index int64, piece_index int64, length int64) *Chunk {
    c := Chunk{}
    c.index = index
    c.piece_index = piece_index
    c.data = make([]byte, length)

    c.status = ChunkStatusReady

    c.status_lock = &sync.Mutex{}

    return &c
}

func (ch *Chunk) GetStatus() int {
    ch.status_lock.Lock()
    status := ch.status
    ch.status_lock.Unlock()

    return status
}

func (ch *Chunk) SetStatus(status int) {
    ch.status_lock.Lock()
    ch.status = status
    ch.status_lock.Unlock()
}

func (ch *Chunk) GetData() []byte {
    return ch.data[:]
}

func (ch *Chunk) SetData(data []byte) {
    ch.data = data[:]
}

func (ch *Chunk) GetIndex() int64 {
    return ch.index
}

func (ch *Chunk) GetPieceIndex() int64 {
    return ch.piece_index
}

func (ch *Chunk) GetLength() int64 {
    return int64(len(ch.data))
}