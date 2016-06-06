package main

import (
	"fmt"
	"image/gif"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net/http"

	"golang.org/x/image/bmp"
	"golang.org/x/image/webp"
)

func bmpFile(w http.ResponseWriter, r *http.Request, f multipart.File, file File) (_ File, err error) {
	if size, err := fileSize(f); err == nil {
		file.Size = size
	} else {
		return file, err
	}

	if imgConf, err := bmp.DecodeConfig(f); err == nil {
		file.Width = imgConf.Width
		file.Height = imgConf.Height
		f.Seek(0, 0)
	} else {
		return file, err
	}

	random := RandomFilename()
	file.File = fmt.Sprintf("%s.bmp", random)
	file.Thumb = fmt.Sprintf("%s_t.jpg", random)

	if err = file.UploadFile(f); err != nil {
		return file, err
	}

	if img, err := bmp.Decode(f); err == nil {
		if imgConf, err := file.UploadThumbnail(img); err == nil {
			file.ThumbWidth = imgConf.Width
			file.ThumbHeight = imgConf.Height
		}
	}

	return file, err
}

func gifFile(w http.ResponseWriter, r *http.Request, f multipart.File, file File) (_ File, err error) {
	if size, err := fileSize(f); err == nil {
		file.Size = size
	} else {
		return file, err
	}

	if imgConf, err := gif.DecodeConfig(f); err == nil {
		file.Width = imgConf.Width
		file.Height = imgConf.Height
		f.Seek(0, 0)
	} else {
		return file, err
	}

	random := RandomFilename()
	file.File = fmt.Sprintf("%s.jpg", random)
	file.Thumb = fmt.Sprintf("%s_t.jpg", random)

	if err = file.UploadFile(f); err != nil {
		return file, err
	}

	if img, err := gif.Decode(f); err == nil {
		if imgConf, err := file.UploadThumbnail(img); err == nil {
			file.ThumbWidth = imgConf.Width
			file.ThumbHeight = imgConf.Height
		}
	}

	return file, err
}


func jpegFile(w http.ResponseWriter, r *http.Request, f multipart.File, file File) (_ File, err error) {
	if size, err := fileSize(f); err == nil {
		file.Size = size
	} else {
		return file, err
	}

	if imgConf, err := jpeg.DecodeConfig(f); err == nil {
		file.Width = imgConf.Width
		file.Height = imgConf.Height
		f.Seek(0, 0)
	} else {
		return file, err
	}

	random := RandomFilename()
	file.File = fmt.Sprintf("%s.jpg", random)
	file.Thumb = fmt.Sprintf("%s_t.jpg", random)

	if err = file.UploadFile(f); err != nil {
		return file, err
	}

	if img, err := jpeg.Decode(f); err == nil {
		if imgConf, err := file.UploadThumbnail(img); err == nil {
			file.ThumbWidth = imgConf.Width
			file.ThumbHeight = imgConf.Height
		}
	}

	return file, err
}

func pngFile(w http.ResponseWriter, r *http.Request, f multipart.File, file File) (_ File, err error) {
	if size, err := fileSize(f); err == nil {
		file.Size = size
	} else {
		return file, err
	}

	if imgConf, err := png.DecodeConfig(f); err == nil {
		file.Width = imgConf.Width
		file.Height = imgConf.Height
		f.Seek(0, 0)
	} else {
		return file, err
	}

	random := RandomFilename()
	file.File = fmt.Sprintf("%s.png", random)
	file.Thumb = fmt.Sprintf("%s_t.jpg", random)

	if err = file.UploadFile(f); err != nil {
		return file, err
	}

	if img, err := png.Decode(f); err == nil {
		if imgConf, err := file.UploadThumbnail(img); err == nil {
			file.ThumbWidth = imgConf.Width
			file.ThumbHeight = imgConf.Height
		}
	}

	return file, err
}

func webpFile(w http.ResponseWriter, r *http.Request, f multipart.File, file File) (_ File, err error) {
	if size, err := fileSize(f); err == nil {
		file.Size = size
	} else {
		return file, err
	}

	if imgConf, err := webp.DecodeConfig(f); err == nil {
		file.Width = imgConf.Width
		file.Height = imgConf.Height
		f.Seek(0, 0)
	} else {
		return file, err
	}

	random := RandomFilename()
	file.File = fmt.Sprintf("%s.webp", random)
	file.Thumb = fmt.Sprintf("%s_t.jpg", random)

	if err = file.UploadFile(f); err != nil {
		return file, err
	}

	if img, err := webp.Decode(f); err == nil {
		if imgConf, err := file.UploadThumbnail(img); err == nil {
			file.ThumbWidth = imgConf.Width
			file.ThumbHeight = imgConf.Height
		}
	}

	return file, err
}
