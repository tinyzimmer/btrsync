package sendstream

import "errors"

var (
	ErrInvalidMagic           = errors.New("invalid magic")
	ErrInvalidVersion         = errors.New("invalid version")
	ErrHeaderAlreadyParsed    = errors.New("header already parsed")
	ErrInvalidCommandChecksum = errors.New("invalid crc32 checksum for command")
)
