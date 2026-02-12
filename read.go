package gombus

import (
	"bufio"
	"fmt"
	"io"
	"time"
)

var ErrNoLongFrameFound = fmt.Errorf("no long frame found")

func ReadLongFrame(conn Conn, timeout time.Duration) (LongFrame, error) {
	buf := make([]byte, 4096)
	tmp := make([]byte, 4096)

	// foundStart := false
	length := 0
	globalN := -1
	for {
		err := conn.SetReadDeadline(time.Now().Add(timeout))
		if err != nil {
			return LongFrame{}, fmt.Errorf("error from SetReadDeadline: %w", err)
		}

		n, err := conn.Read(tmp)
		if err != nil {
			return LongFrame{}, fmt.Errorf("error reading from connection: %w", err)
		}

		for _, b := range tmp[:n] {
			globalN++
			buf[globalN] = b

			if globalN > 256 {
				return LongFrame{}, ErrNoLongFrameFound
			}

			// look for end byte after length +C+A+CI+checksum
			if length != 0 && globalN > length+4 && b == 0x16 {
				return LongFrame(buf[:globalN+1]), nil
			}

			// look for start sequence 68 LL LL 68
			if length == 0 && buf[0] == 0x68 && buf[3] == 0x68 && buf[1] == buf[2] {
				length = int(buf[1])
			}
		}
	}
}

func ReadAckFrame(r io.Reader) error {
	buf := bufio.NewReader(r)
	_, err := buf.ReadBytes(SingleCharacterFrame)

	return err
}
