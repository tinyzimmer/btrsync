package btrfs

import (
	"bytes"
	"compress/zlib"
	"io"

	"github.com/klauspost/compress/zstd"
	"github.com/rasky/go-lzo"
)

func decompressZlip(data []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func decompressLzo(data []byte, encodedLen, unencodedLen int) ([]byte, error) {
	return lzo.Decompress1X(bytes.NewReader(data), encodedLen, unencodedLen)
}

func decompressZstd(data []byte) ([]byte, error) {
	r, err := zstd.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}
