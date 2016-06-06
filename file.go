package main

import (
	"errors"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"
)

const MAX_SIZE = 1024 * 1024 * 10

type FileRouter struct {
	Handlers map[string]func(http.ResponseWriter, *http.Request, multipart.File, File) (File, error)}

func NewFileRouter() (f FileRouter) {
	f.Handlers = make(map[string]func(http.ResponseWriter, *http.Request, multipart.File, File) (File, error))
	return f
}

func (f *FileRouter) AddFile(t string, fn func(http.ResponseWriter, *http.Request, multipart.File, File) (File, error)) {
	f.Handlers[t] = fn
}

func (f *FileRouter) File(w http.ResponseWriter, file multipart.File) (func(http.ResponseWriter, *http.Request, multipart.File, *multipart.FileHeader) (File, error), error) {
	// Slice the first 512 possible bytes from the upload and determine a MIME type
	sample := make([]byte, 512)

	if _, err := file.Read(sample); err != nil {
		return nil, err
	}

	// I think metadata would be useful down the road..
	t, /* metadata */ _, err := mime.ParseMediaType(http.DetectContentType(sample))

	if err != nil {
		return nil, err
	}

	size, err := fileSize(file)

	if err != nil {
		return nil, err
	} else if size > MAX_SIZE {
		return nil, errors.New("File is too large")
	}

	return func(w http.ResponseWriter, r *http.Request, f_ multipart.File, h *multipart.FileHeader) (file File, err error) {
		var count uint

		if handler, ok := f.Handlers[t]; ok {
			file.Name = h.Filename
			file.Type = t
			file.CreateHash(f_)

			if !config.AllowDuplicates {
				db.Model(&File{}).Where("hash = ?", file.Hash).Count(&count)

				if count > 0 {
					err = errors.New("Duplicate detected")
					return
				}
			}

			file, err = handler(w, r, f_, file)
		} else {
			err = errors.New("File type not recognized")
		}

		return
	}, nil
}

// File utilities
func fileSize(file multipart.File) (int, error) {
	limit := io.LimitReader(file, MAX_SIZE + 1)
	data, err := ioutil.ReadAll(limit) // (this is clever, we only read MAX_SIZE + 1 bytes so we don't suffer from a HUGE upload)
	file.Seek(0, 0)
	return len(data), err
}

// Useful psuedorandom filename generatr, shortened to base36
func RandomFilename() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}
