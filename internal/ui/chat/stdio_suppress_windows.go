//go:build windows

package chat

func suppressStdIO() (restore func(), err error) {
	return func() {}, nil
}
