package file

import (
	"fmt"
	"log"
	"os"
	"strings"
)

type File struct {
	Length      int64
	Start_piece int64
	End_piece   int64
	path        []string
	file_handle *os.File
}

func NewFile(length int64, path []string) *File {
	f := File{}
	f.Length = length
	f.Start_piece = 0
	f.End_piece = 0
	f.path = path
	f.file_handle = nil

	return &f
}

func (f *File) GetPath() []string {
	return f.path
}

func (f *File) Write(data []byte, pos int64) {
	if f.file_handle == nil {
		// create folders if needed

		file_path := fmt.Sprintf("./%s", strings.Join(f.GetPath(), "/"))
		fmt.Println(file_path)
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
