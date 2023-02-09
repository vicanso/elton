// MIT License

// Copyright (c) 2022 Tree Xie

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package elton

import (
	"errors"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

type multipartForm struct {
	w           *multipart.Writer
	tmpfile     string
	contentType string
}

// NewMultipartForm returns a new multipart form
func NewMultipartForm() *multipartForm {
	return &multipartForm{}
}

func (f *multipartForm) newFileBuffer() error {
	if f.w != nil {
		return nil
	}
	file, err := os.CreateTemp("", "multipart-form-")
	if err != nil {
		return err
	}
	f.tmpfile = file.Name()
	f.w = multipart.NewWriter(file)
	f.contentType = f.w.FormDataContentType()
	return nil
}

// AddField adds a field to form
func (f *multipartForm) AddField(name, value string) error {
	err := f.newFileBuffer()
	if err != nil {
		return err
	}
	return f.w.WriteField(name, value)
}

// AddFile add a file to form, if the reader is nil, the filename will be used to open as reader
func (f *multipartForm) AddFile(name, filename string, reader ...io.Reader) error {
	err := f.newFileBuffer()
	if err != nil {
		return err
	}
	var r io.Reader
	if len(reader) != 0 {
		r = reader[0]
	} else {
		file, err := os.Open(filename)
		if err != nil {
			return err
		}
		// 调整filename
		filename = filepath.Base(filename)
		defer file.Close()
		r = file
	}

	fw, err := f.w.CreateFormFile(name, filename)
	if err != nil {
		return err
	}
	_, err = io.Copy(fw, r)
	if err != nil {
		return err
	}
	return nil
}

// Reader returns a render of form
func (f *multipartForm) Reader() (io.Reader, error) {
	if f.w == nil {
		return nil, errors.New("multi part is nil")
	}
	err := f.w.Close()
	if err != nil {
		return nil, err
	}
	f.w = nil
	return os.Open(f.tmpfile)
}

// Destroy closes the writer and removes the tmpfile
func (f *multipartForm) Destroy() error {
	if f.w != nil {
		err := f.w.Close()
		if err != nil {
			return err
		}
	}
	if f.tmpfile != "" {
		return os.Remove(f.tmpfile)
	}
	return nil
}

// ContentType returns the content type of form
func (f *multipartForm) ContentType() string {
	return f.contentType
}
