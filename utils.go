package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/common-nighthawk/go-figure"
	"github.com/google/uuid"
)

type DataURL struct {
	MIME string
	Data []byte
}

func AsciiArt() {
	myFigure := figure.NewFigure("1-SDT", "doom", true)
	myFigure.Print()

	fmt.Println("\n\t\t\tHayzam Sherif\n")
}

func CreateLicense(owner string) string {
	year := time.Now().Year()
	return fmt.Sprintf(`MIT License

Copyright (c) %d %s

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.`, year, owner)
}

func ToBase64(data string) string {
	return base64.StdEncoding.EncodeToString([]byte(data))
}

func GenerateUUID() string {
	return uuid.NewString()
}

func StringArrToString(arr []string) string {
	result := ""
	for i, str := range arr {
		if i == 0 {
			result += str
		} else {
			result += "\n" + str
		}
	}
	return result
}

func DecodeDataURL(s string) (DataURL, error) {
	if !strings.HasPrefix(s, "data:") {
		return DataURL{}, errors.New("not a data: URL")
	}

	parts := strings.SplitN(s, ",", 2)
	if len(parts) != 2 {
		return DataURL{}, errors.New("invalid data URL")
	}

	meta, b64 := parts[0], parts[1]

	if !strings.HasSuffix(meta, ";base64") {
		return DataURL{}, errors.New("only base64 data URLs supported")
	}

	mime := strings.TrimPrefix(strings.TrimSuffix(meta, ";base64"), "data:")
	raw, err := base64.StdEncoding.DecodeString(b64)

	if err != nil {
		return DataURL{}, err
	}

	return DataURL{MIME: mime, Data: raw}, nil
}

func ToBase64Bytes(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

func FromBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}
