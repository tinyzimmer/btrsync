package btrfs

import (
	"os"
	"unsafe"
)

func EnableVerity(path string, algorithm uint32, blockSize uint32, salt []byte, sig []byte) error {
	return enableVerity(path, algorithm, blockSize, salt, sig)
}

func enableVerity(path string, algorithm uint32, blockSize uint32, salt []byte, sig []byte) error {
	args := fsVerityEnableArg{
		Version:        1,
		Hash_algorithm: algorithm,
		Block_size:     blockSize,
		Salt_size:      uint32(len(salt)),
		Salt_ptr:       uint64(uintptr(unsafe.Pointer(&salt[0]))),
		Sig_size:       uint32(len(sig)),
		Sig_ptr:        uint64(uintptr(unsafe.Pointer(&sig[0]))),
	}
	f, err := os.OpenFile(path, os.O_RDONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	return callWriteIoctl(f.Fd(), FS_IOC_ENABLE_VERITY, &args)
}
