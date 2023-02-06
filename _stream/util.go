package _stream

func Drain[T any](stream <-chan T) []T {
	rs := make([]T, 0)

	for v := range stream {
		rs = append(rs, v)
	}

	return rs
}

func OfSlice[T any](cacheSize int, rs ...T) <-chan T {
	stream := make(chan T, cacheSize)
	go func() {
		defer close(stream)

		for _, v := range rs {
			stream <- v
		}
	}()

	return stream
}
