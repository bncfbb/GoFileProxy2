package io

import "io"

func Copy(writer io.Writer, reader io.Reader, buffersize int64) (written int64, err error) {
	var total int64

	for {
		buffer := make([]byte, buffersize)

		len, err := reader.Read(buffer)
		if err != nil {
			if err == io.EOF {
				writer.Write(buffer[:len])
				total += int64(len)
			}
			//log.Fatal(err)
			return total, err
		}
		if len == 0 {
			break
		}
		if _, err := writer.Write(buffer[:len]); err != nil {
			return total, err
		}
		total += int64(len)
	}
	return total, nil
}
