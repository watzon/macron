// Copyright (c) 2024 Chris Watson
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package styling

import (
	"errors"
	"io"
	"strings"
	"unicode/utf8"
)

// A Reader implements the [io.Reader], [io.ReaderAt], [io.ByteReader], [io.ByteScanner],
// [io.RuneReader], [io.RuneScanner], [io.Seeker], and [io.WriterTo] interfaces by reading
// from a string.
// The zero value for Reader operates like a Reader of an empty string.
type Reader struct {
	s        string
	i        int64 // current reading index
	prevRune int   // index of previous rune; or < 0
}

// Len returns the number of bytes of the unread portion of the
// string.
func (r *Reader) Len() int {
	if r.i >= int64(len(r.s)) {
		return 0
	}
	return int(int64(len(r.s)) - r.i)
}

// Size returns the original length of the underlying string.
// Size is the number of bytes available for reading via [Reader.ReadAt].
// The returned value is always the same and is not affected by calls
// to any other method.
func (r *Reader) Size() int64 { return int64(len(r.s)) }

// Read implements the [io.Reader] interface.
func (r *Reader) Read(b []byte) (n int, err error) {
	if r.i >= int64(len(r.s)) {
		return 0, io.EOF
	}
	r.prevRune = -1
	n = copy(b, r.s[r.i:])
	r.i += int64(n)
	return
}

// ReadAt implements the [io.ReaderAt] interface.
func (r *Reader) ReadAt(b []byte, off int64) (n int, err error) {
	// cannot modify state - see io.ReaderAt
	if off < 0 {
		return 0, errors.New("styling.Reader.ReadAt: negative offset")
	}
	if off >= int64(len(r.s)) {
		return 0, io.EOF
	}
	n = copy(b, r.s[off:])
	if n < len(b) {
		err = io.EOF
	}
	return
}

// ReadUntil reads until the given string is found, returning the data read and the
// string if found. The returned byte slice is only valid until the next read.
func (r *Reader) ReadUntil(s string) (string, error) {
	i := strings.Index(r.s[r.i:], s)
	if i < 0 {
		return "", io.EOF
	}
	return r.s[r.i : r.i+int64(i)], nil
}

// ReadLine reads until the first occurrence of delim, returning a string containing
// the data read up to and including the delimiter. If ReadLine encounters an error
// before finding a delimiter, it returns the data read before the error and the error
// encountered.
func (r *Reader) ReadLine() string {
	i := strings.Index(r.s[r.i:], "\n")
	if i < 0 {
		return r.s[r.i:]
	}
	return r.s[r.i : r.i+int64(i)]
}

// ReadByte implements the [io.ByteReader] interface.
func (r *Reader) ReadByte() (byte, error) {
	r.prevRune = -1
	if r.i >= int64(len(r.s)) {
		return 0, io.EOF
	}
	b := r.s[r.i]
	r.i++
	return b, nil
}

// UnreadByte implements the [io.ByteScanner] interface.
func (r *Reader) UnreadByte() error {
	if r.i <= 0 {
		return errors.New("styling.Reader.UnreadByte: at beginning of string")
	}
	r.prevRune = -1
	r.i--
	return nil
}

// ReadRune implements the [io.RuneReader] interface.
func (r *Reader) ReadRune() (ch rune, size int, err error) {
	if r.i >= int64(len(r.s)) {
		r.prevRune = -1
		return 0, 0, io.EOF
	}
	r.prevRune = int(r.i)
	if c := r.s[r.i]; c < utf8.RuneSelf {
		r.i++
		return rune(c), 1, nil
	}
	ch, size = utf8.DecodeRuneInString(r.s[r.i:])
	r.i += int64(size)
	return
}

// UnreadRune implements the [io.RuneScanner] interface.
func (r *Reader) UnreadRune() error {
	if r.i <= 0 {
		return errors.New("styling.Reader.UnreadRune: at beginning of string")
	}
	if r.prevRune < 0 {
		return errors.New("styling.Reader.UnreadRune: previous operation was not ReadRune")
	}
	r.i = int64(r.prevRune)
	r.prevRune = -1
	return nil
}

// Seek implements the [io.Seeker] interface.
func (r *Reader) Seek(offset int64, whence int) (int64, error) {
	r.prevRune = -1
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = r.i + offset
	case io.SeekEnd:
		abs = int64(len(r.s)) + offset
	default:
		return 0, errors.New("styling.Reader.Seek: invalid whence")
	}
	if abs < 0 {
		return 0, errors.New("styling.Reader.Seek: negative position")
	}
	r.i = abs
	return abs, nil
}

// WriteTo implements the [io.WriterTo] interface.
func (r *Reader) WriteTo(w io.Writer) (n int64, err error) {
	r.prevRune = -1
	if r.i >= int64(len(r.s)) {
		return 0, nil
	}
	s := r.s[r.i:]
	m, err := io.WriteString(w, s)
	if m > len(s) {
		panic("styling.Reader.WriteTo: invalid WriteString count")
	}
	r.i += int64(m)
	n = int64(m)
	if m != len(s) && err == nil {
		err = io.ErrShortWrite
	}
	return
}

// Peek returns the next N bytes as a string without advancing the reader. The returned
// byte slice is only valid until the next read.
func (r *Reader) Peek(n int) (string, error) {
	if r.i >= int64(len(r.s)) {
		return "", io.EOF
	}
	if n > len(r.s) {
		return "", errors.New("styling.Reader.Peek: n > len(s)")
	}
	return r.s[r.i : r.i+int64(n)], nil
}

// PeekByte returns the next byte without advancing the reader. If no byte is available,
// it returns error.
func (r *Reader) PeekByte() (byte, error) {
	if r.i >= int64(len(r.s)) {
		return 0, io.EOF
	}
	return r.s[r.i], nil
}

// PeekRune returns the next rune without advancing the reader. If no more runes are
// available, it returns error.
func (r *Reader) PeekRune() (rune, int, error) {
	if r.i >= int64(len(r.s)) {
		return 0, 0, io.EOF
	}
	r.prevRune = -1
	if c := r.s[r.i]; c < utf8.RuneSelf {
		return rune(c), 1, nil
	}
	ch, size := utf8.DecodeRuneInString(r.s[r.i:])
	return ch, size, nil
}

// Skip skips the next n bytes. It returns the number of bytes skipped and an error if
// any occurred.
func (r *Reader) Skip(n int) (int, error) {
	if n < 0 {
		return 0, errors.New("styling.Reader.Skip: negative n")
	}
	if n > len(r.s) {
		return 0, errors.New("styling.Reader.Skip: n > len(s)")
	}
	r.prevRune = -1
	r.i += int64(n)
	return n, nil
}

// SetIndex sets the current reading index to the given position.
func (r *Reader) SetIndex(i int64) error {
	if i < 0 {
		return errors.New("styling.Reader.SetIndex: negative index")
	}
	if i > int64(len(r.s)) {
		return errors.New("styling.Reader.SetIndex: index out of range")
	}
	r.prevRune = -1
	r.i = i
	return nil
}

// Reset resets the [Reader] to be reading from s.
func (r *Reader) Reset(s string) { *r = Reader{s, 0, -1} }

// NewReader returns a new [Reader] reading from s.
// It is similar to [bytes.NewBufferString] but more efficient and non-writable.
func NewReader(s string) *Reader { return &Reader{s, 0, -1} }
