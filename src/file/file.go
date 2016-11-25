package file

import (
	"fmt"
	"log"
	"os"
	"strings"
	"path/filepath"
)

type File struct {
	start_piece  int64
	end_piece    int64
	length       int64
	path         []string
	downloadable bool

	file_handle  *os.File
}

func NewFile(length int64, path []string) *File {
	f := File{}
	f.length = length
	f.path = path

	return &f
}

func (f *File) SetStartPiece(start_piece int64) {
	f.start_piece = start_piece
}

func (f *File) SetEndPiece(end_piece int64) {
	f.end_piece = end_piece
}

func (f *File) SetDownloadable(downloadable bool) {
	f.downloadable = downloadable
}

func (f *File) GetStartAndEndPieces() (int64, int64) {
	return f.start_piece, f.end_piece
}

func (f *File) IsDownloadable() bool {
	return f.downloadable
}

func (f *File) GetPath() []string {
	return f.path
}

func (f *File) GetDisplayPath() []string {
	return f.path[2:]
}

func (f *File) GetLength() int64 {
	return f.length
}

func (f *File) Write(data []byte, pos int64) {
	if f.IsDownloadable() == false {
		return
	}
	
	if f.file_handle == nil {
		// create folders if needed
		path := f.GetPath()
		file_path := fmt.Sprintf("%s", strings.Join(path, "/"))
		folder_path := fmt.Sprintf("%s", strings.Join(path[0:len(path)-1], "/"))
		os.MkdirAll(filepath.Join(folder_path), os.ModePerm)

		fh, err := os.OpenFile(
			file_path,
			os.O_WRONLY|os.O_SYNC|os.O_CREATE,
			0666,
		)
		f.file_handle = fh

		if err != nil {
			log.Fatal(err)
		}
	}

	var whence int = 0
	_, err := f.file_handle.Seek(pos, whence)
	if err != nil {
		log.Fatal(err)
	}

	// Write bytes to file
	n, err := f.file_handle.Write(data)
	if err != nil || n != len(data) {
		log.Fatal(err)
	}
}

func (f *File) Close() {
	if f.file_handle != nil {
		f.file_handle.Close()
	}
}
