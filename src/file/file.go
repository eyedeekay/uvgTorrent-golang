package file

type File struct {
    Length  int64
    Start_piece int64
    End_piece int64
    path    []string
}

func NewFile(length int64, path []string) *File {
    f := File{}
    f.Length = length
    f.Start_piece = 0
    f.End_piece = 0
    f.path = path
    
    return &f
}

func (f *File) GetPath() []string {
    return f.path
}