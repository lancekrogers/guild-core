//go:build !windows

package chat

import (
	"os"

	"golang.org/x/sys/unix"
)

func suppressStdIO() (restore func(), err error) {
	nullFile, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		return func() {}, err
	}

	stdoutFD := int(os.Stdout.Fd())
	stderrFD := int(os.Stderr.Fd())

	savedStdoutFD, err := unix.Dup(stdoutFD)
	if err != nil {
		_ = nullFile.Close()
		return func() {}, err
	}

	savedStderrFD, err := unix.Dup(stderrFD)
	if err != nil {
		_ = unix.Close(savedStdoutFD)
		_ = nullFile.Close()
		return func() {}, err
	}

	restore = func() {
		_ = unix.Dup2(savedStdoutFD, stdoutFD)
		_ = unix.Dup2(savedStderrFD, stderrFD)
		_ = unix.Close(savedStdoutFD)
		_ = unix.Close(savedStderrFD)
		_ = nullFile.Close()
	}

	nullFD := int(nullFile.Fd())
	if err := unix.Dup2(nullFD, stdoutFD); err != nil {
		restore()
		return func() {}, err
	}
	if err := unix.Dup2(nullFD, stderrFD); err != nil {
		restore()
		return func() {}, err
	}

	return restore, nil
}
