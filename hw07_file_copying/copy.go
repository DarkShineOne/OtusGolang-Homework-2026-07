package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/cheggaaa/pb" ////nolint:depguard
)

var (
	ErrUnsupportedFile       = errors.New("unsupported file")
	ErrOffsetExceedsFileSize = errors.New("offset exceeds file size")
	ErrInvalidArgs           = errors.New("invalid arguments")
	ErrSameFile              = errors.New("source and destination are the same file")
)

const bufSize = 1024

var progressOut io.Writer = os.Stdout

func Copy(fromPath, toPath string, offset, limit int64) error {
	if offset < 0 || limit < 0 {
		return fmt.Errorf("%w: offset=%d limit=%d", ErrInvalidArgs, offset, limit)
	}

	stat, err := os.Stat(fromPath)
	if err != nil {
		return err
	}

	if !stat.Mode().IsRegular() {
		return ErrUnsupportedFile
	}

	if offset > stat.Size() {
		return ErrOffsetExceedsFileSize
	}

	if toStat, statErr := os.Stat(toPath); statErr == nil && os.SameFile(stat, toStat) {
		return ErrSameFile
	}

	inputF, err := os.Open(fromPath)
	if err != nil {
		return err
	}
	defer inputF.Close()

	if _, err = inputF.Seek(offset, io.SeekStart); err != nil {
		return err
	}

	total := stat.Size() - offset
	if limit > 0 && limit < total {
		total = limit
	}

	outputF, err := os.OpenFile(toPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, stat.Mode().Perm())
	if err != nil {
		return err
	}

	bar := pb.New(100)
	bar.Output = progressOut
	bar.Start()
	defer bar.Finish()

	buf := make([]byte, bufSize)

	var copied int64
	for copied < total {
		read, readErr := inputF.Read(buf)
		if read > 0 {
			if copied+int64(read) > total {
				read = int(total - copied)
			}
			if _, err = outputF.Write(buf[:read]); err != nil {
				outputF.Close()
				return err
			}
			copied += int64(read)
			bar.Set(percent(copied, total))
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			outputF.Close()
			return readErr
		}
		if read == 0 {
			return io.ErrUnexpectedEOF
		}
	}

	if err := outputF.Close(); err != nil {
		return err
	}

	return nil
}

func percent(copied, total int64) int {
	if total <= 0 {
		return 100
	}
	return int(float64(copied) * 100 / float64(total))
}
