/**
	RESP protocol reader
 */
package protocol

import (
	"bufio"
	"io"
	"fmt"
	"errors"
	"strconv"
	"bytes"
)

const (
	SIMPLE_STRING = '+'
	BULK_STRING   = '$'
	INTEGER       = ':'
	ARRAY         = '*'
	ERROR         = '-'
)

var (
	ErrInvalidSyntax = errors.New("resp: invalid syntax")
)

type RESPReader struct {
	src *bufio.Reader
}

func NewRESPReader(reader io.Reader) *RESPReader {
	return &RESPReader{
		src: bufio.NewReader(reader),
	}
}

func (r *RESPReader) ReadObject() ([]byte, error) {
	line, err := r.readLine()
	if err != nil {
		return nil, err
	}
	switch line[0] {
	case SIMPLE_STRING, INTEGER, ERROR:
		return line, nil
	case BULK_STRING:
		return r.readBulkString(line)
	case ARRAY:
		return r.readArray(line)
	default:
		return nil, ErrInvalidSyntax
	}
}

// 解析本文长度
func (r *RESPReader) getCount(line []byte) (int, error) {
	end := bytes.IndexByte(line, '\r')
	if end == -1 {
		return 0, fmt.Errorf("proto err")
	}
	return strconv.Atoi(string(line[1:end]))
}

func (r *RESPReader) readArray(line []byte) ([]byte, error) {
	count, err := r.getCount(line)
	if err != nil {
		return nil, err
	}
	for i := 0; i < count; i++ {
		buf, err := r.ReadObject()
		if err != nil {
			return nil, err
		}
		line = append(line, buf...)
	}
	return line, nil
}

func (r *RESPReader) readBulkString(line []byte) ([]byte, error) {
	count, err := r.getCount(line)
	if err != nil {
		return nil, err
	}
	if count == -1 {
		return line, nil
	}
	buf := make([]byte, len(line)+count+2)
	copy(buf, line)
	_, err = io.ReadFull(r.src, buf[len(line):])
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func (r *RESPReader) readLine() ([]byte, error) {
	line, err := r.src.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	if len(line) > 1 && line[len(line)-2] == '\r' {
		return line, nil
	}
	return nil, ErrInvalidSyntax

	}
