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

package sendstream

import (
	"encoding/binary"
	"fmt"
	"io"
)

var ErrHeaderAlreadySent = fmt.Errorf("header already sent")

type Writer struct {
	io.Writer
	headerSent bool
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{Writer: w}
}

func (w *Writer) SendHeader() error {
	if w.headerSent {
		return ErrHeaderAlreadySent
	}
	if err := w.write(&StreamHeader{
		Magic:   BTRFS_SEND_STREAM_MAGIC_ENCODED,
		Version: BTRFS_SEND_STREAM_VERSION,
	}); err != nil {
		return err
	}
	w.headerSent = true
	return nil
}

func (w *Writer) WriteCommand(cmd SendCommand, attrs CmdAttrs) error {
	if !w.headerSent {
		if err := w.SendHeader(); err != nil {
			return err
		}
	}
	cmdHeader := CmdHeader{
		Cmd: cmd,
		Len: attrs.BinarySize(),
	}
	data, err := attrs.Encode()
	if err != nil {
		return err
	}
	cmdHeader.Crc, err = calculateCrc32(cmdHeader, data)
	if err != nil {
		return err
	}
	if err := w.write(&cmdHeader); err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	return nil
}

func (w *Writer) write(data any) error { return binary.Write(w, binary.LittleEndian, data) }
