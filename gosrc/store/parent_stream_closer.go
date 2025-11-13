package store

import "io"

type CloseWrapper struct {
	openReaderRef io.Reader
	openStreamRef io.Closer
}

/*Create a new CloseWrapper which wraps a reader with no closer and the parent stream that is being read from.*/
func NewCloseWrapper(reader io.Reader, closer io.Closer) *CloseWrapper {
	return &CloseWrapper{
		openReaderRef: reader,
		openStreamRef: closer,
	}
}

func (cw *CloseWrapper) Read(p []byte) (n int, err error) {
	return cw.openReaderRef.Read(p)
}

func (cw *CloseWrapper) Close() error {
	return cw.openStreamRef.Close()
}
