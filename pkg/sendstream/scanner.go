/*
This file is part of btrsync.

Btrsync is free software: you can redistribute it and/or modify it under the terms of the
GNU Lesser General Public License as published by the Free Software Foundation, either
version 3 of the License, or (at your option) any later version.

Btrsync is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY;
without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
See the GNU Lesser General Public License for more details.

You should have received a copy of the GNU Lesser General Public License along with btrsync.
If not, see <https://www.gnu.org/licenses/>.
*/

// Pacakge sendstream implements a scanner for the btrfs send stream format.
package sendstream

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// Scanner is a send stream scanner. It reads a send stream from an io.Reader
// and parses it into commands. It is not safe for concurrent use.
type Scanner struct {
	io.Reader
	ignoreChecksums bool
	headerParsed    bool
	scanErr         error
	curHdr          CmdHeader
	curAttrs        CmdAttrs
}

// NewScanner returns a new Scanner that reads from r. If ignoreChecksums is
// true, the scanner will ignore crc32 checksum errors.
func NewScanner(r io.Reader, ignoreChecksums bool) *Scanner {
	return &Scanner{Reader: r, ignoreChecksums: ignoreChecksums}
}

// Scan advances the scanner to the next command. It returns false when the
// scan stops, either by reaching the end of the input or an error. After Scan
// returns false, the Err method will return any error that occurred during
// scanning, except that if it was io.EOF after an END command, Err will return nil.
// If the header has not been parsed yet, Scan will parse it before reading the
// first command.
func (s *Scanner) Scan() bool {
	if s.scanErr != nil {
		return false
	}
	if !s.headerParsed {
		if _, err := s.readHeader(); err != nil {
			s.scanErr = err
			return false
		}
	}
	hdr, attrs, err := s.ReadCommand()
	if err != nil {
		if s.curHdr.IsZero() {
			s.scanErr = err
			return false
		}
		if s.curHdr.Cmd == BTRFS_SEND_C_END {
			if errors.Is(err, io.EOF) {
				s.scanErr = nil
				return false
			}
		}
		s.scanErr = err
		return false
	}
	s.curHdr = hdr
	s.curAttrs = attrs
	return true
}

// Command returns the most recent command generated by a call to Scan.
func (s *Scanner) Command() (CmdHeader, CmdAttrs) { return s.curHdr, s.curAttrs }

// Err returns the first non-EOF/non-END error that was encountered by the Scanner.
func (s *Scanner) Err() error { return s.scanErr }

// ReadHeader reads the stream header from r. It returns an error if the header
// is invalid or has already been parsed. If validate is false, the magic and version
// are not validated.
func (s *Scanner) ReadHeader(validate bool) (StreamHeader, error) {
	if s.headerParsed {
		return StreamHeader{}, ErrHeaderAlreadyParsed
	}
	hdr, err := s.readHeader()
	if err != nil {
		if errors.Is(err, ErrInvalidMagic) || errors.Is(err, ErrInvalidVersion) {
			if !validate {
				return hdr, nil
			}
		}
		return StreamHeader{}, err
	}
	return hdr, nil
}

// ReadCommand reads the next command from r.
func (s *Scanner) ReadCommand() (CmdHeader, CmdAttrs, error) {
	hdr, err := s.readCommandHeader()
	if err != nil {
		return CmdHeader{}, nil, err
	}
	attrs, err := s.readCommandAttributes(hdr)
	return hdr, attrs, err
}

func (s *Scanner) readHeader() (StreamHeader, error) {
	var hdr StreamHeader
	if err := s.read(&hdr); err != nil {
		return hdr, err
	}
	defer func() { s.headerParsed = true }()
	if string(hdr.Magic[:]) != BTRFS_SEND_STREAM_MAGIC {
		return hdr, fmt.Errorf("%w %q", ErrInvalidMagic, hdr.Magic)
	}
	if hdr.Version != BTRFS_SEND_STREAM_VERSION {
		return hdr, fmt.Errorf("%w %d", ErrInvalidVersion, hdr.Version)
	}
	return hdr, nil
}

func (s *Scanner) readCommandHeader() (CmdHeader, error) {
	var hdr CmdHeader
	if err := s.read(&hdr); err != nil {
		return CmdHeader{}, err
	}
	return hdr, nil
}

func (s *Scanner) readCommandAttributes(hdr CmdHeader) (CmdAttrs, error) {
	size := int(hdr.Len)
	attrs := make(CmdAttrs)
	data := make([]byte, size)
	if err := s.read(data); err != nil {
		return nil, err
	}
	if !s.ignoreChecksums {
		if err := validateCrc32(hdr, data); err != nil {
			return nil, err
		}
	}
	var pos int
	for pos < size {
		var attr SendAttribute
		if err := binary.Read(bytes.NewReader(data[pos:]), binary.LittleEndian, &attr); err != nil {
			return nil, err
		}
		pos += binary.Size(attr)
		var attrLen uint32
		if attr == BTRFS_SEND_A_DATA {
			attrLen = uint32(size - pos)
		} else {
			var len uint16
			if err := binary.Read(bytes.NewReader(data[pos:]), binary.LittleEndian, &len); err != nil {
				return nil, err
			}
			pos += binary.Size(len)
			attrLen = uint32(len)
		}
		attrs[attr] = data[pos : pos+int(attrLen)]
		pos += int(attrLen)
	}
	return attrs, nil
}

func (s *Scanner) read(out any) error { return binary.Read(s, binary.LittleEndian, out) }
