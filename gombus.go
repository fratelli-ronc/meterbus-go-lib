package gombus

import (
	"time"
)

func RequestUD2(primaryID uint8) ShortFrame {
	data := NewShortFrame()
	data[1] = 0x5b
	data[2] = primaryID
	data.SetChecksum()
	return data
}

// SndNKE slave will ack with SingleCharacterFrame (e5).
func SndNKE(primaryID uint8) ShortFrame {
	data := NewShortFrame()
	data[1] = 0x40
	data[2] = primaryID
	data.SetChecksum()
	return data
}

func ApplicationReset(primaryID uint8) LongFrame {
	data := LongFrame{
		0x68, // Start byte long/control
		0x06, // length
		0x06, // length
		0x68, // Start byte long/control
		0x73, // SND_UD
		primaryID,
		0x50, // CI field data send
		0x00, // checksum
		0x16, // stop byte
	}

	data.SetLength()
	data.SetChecksum()
	return data
}

// water meter has 19004636 7704 14 07.
func RequestPageChange(primaryID uint8, memoryPage uint8) LongFrame {
	data := LongFrame{
		0x68, // Start byte long/control
		0x00, // length
		0x00, // length
		0x68, // Start byte long/control

		0x73, // REQ_UD2
		primaryID,
		0x51, // CI-field data send

		0xF,  // address
		0x2,  // address
		0x78, // address

		memoryPage,

		0x0, // medium code

		0x00, // checksum
		0x16, // stop byte
	}

	data.SetLength()
	data.SetChecksum()

	return data
}

func SetPrimaryUsingSecondary(secondary uint64, primary uint8) LongFrame {
	data := LongFrame{
		0x68, // Start byte long/control
		0x00, // length
		0x00, // length
		0x68, // Start byte long/control
		0x73, // SND_UD
		0xFD,
		0x51, // CI field data send
		0x00, // address
		0x00, // address
		0x00, // address
		0x00, // address
		0xFF, // manufacturer code
		0xFF, // manufacturer code
		0xFF, // id
		0xFF, // medium code
		0x01, // DIF field
		0x7a, // VIF field
		primary,
		0x00, // checksum
		0x16, // stop byte
	}

	a := uintToBCD(secondary, 4)
	data[7] = a[0]
	data[8] = a[1]
	data[9] = a[2]
	data[10] = a[3]

	data.SetLength()
	data.SetChecksum()
	return data
}

func SetPrimaryUsingPrimary(oldPrimary uint8, newPrimary uint8) LongFrame {
	data := LongFrame{
		0x68, // Start byte long/control
		0x06, // length
		0x06, // length
		0x68, // Start byte long/control
		0x73, // REQ_UD2
		oldPrimary,
		0x51, // CI field data send
		0x01, // DIF field
		0x7a, // VIF field
		newPrimary,
		0x00, // checksum
		0x16, // stop byte
	}

	data.SetLength()
	data.SetChecksum()
	return data
}

func SendPageChange(conn Conn, primaryID uint8, memoryPage uint8, timeout time.Duration) error {
	frame := RequestPageChange(primaryID, memoryPage)

	_, err := conn.Write(frame)
	if err != nil {
		return err
	}

	err = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if err != nil {
		return err
	}
	err = ReadAckFrame(conn)
	if err != nil {
		return err
	}

	return nil
}

// ReadAllFrames supports FCB and reads out all frames from the device using primaryID.
func ReadAllFrames(conn Conn, primaryID uint8, timeout time.Duration) ([]*DecodedFrame, error) {
	frame := SndNKE(primaryID)
	_, err := conn.Write(frame)
	if err != nil {
		return nil, err
	}

	err = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if err != nil {
		return nil, err
	}
	err = ReadAckFrame(conn)
	if err != nil {
		return nil, err
	}

	frames := []*DecodedFrame{}
	respFrame := &DecodedFrame{}
	lastFCB := true
	frameCnt := 0
	for respFrame.HasMoreRecords() || frameCnt == 0 {
		frameCnt++
		frame = RequestUD2(uint8(primaryID))
		if !lastFCB {
			frame.SetFCB()
			frame.SetChecksum()
		}
		lastFCB = frame.C().FCB()

		_, err = conn.Write(frame)
		if err != nil {
			return nil, err
		}

		resp, err := ReadLongFrame(conn, timeout)
		if err != nil {
			return nil, err
		}

		respFrame, err = resp.Decode()
		if err != nil {
			return nil, err
		}
		frames = append(frames, respFrame)
	}

	return frames, nil
}

// ReadSingleFrame reads one frame from the device.
func ReadSingleFrame(conn Conn, primaryID uint8, timeout time.Duration) (*DecodedFrame, error) {
	frame := SndNKE(primaryID)
	_, err := conn.Write(frame)
	if err != nil {
		return nil, err
	}

	err = conn.SetReadDeadline(time.Now().Add(timeout))
	if err != nil {
		return nil, err
	}
	err = ReadAckFrame(conn)
	if err != nil {
		return nil, err
	}

	frame = RequestUD2(primaryID)
	if _, err := conn.Write(frame); err != nil {
		return nil, err
	}

	resp, err := ReadLongFrame(conn, timeout)
	if err != nil {
		return nil, err
	}

	respFrame, err := resp.Decode()
	if err != nil {
		return nil, err
	}

	return respFrame, nil
}
