package app

type chanWriter chan ([]byte)

func (c chanWriter) Write(data []byte) (n int, err error) {
	c <- data
	return len(data), nil
}
