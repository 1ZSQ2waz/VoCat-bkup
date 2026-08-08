package modem

import (
	"context"
	"errors"
	"fmt"

	"go.bug.st/serial"
)

type SerialOpener struct {
	SessionOptions SessionOptions
	BaudRate       int
}

func (opener SerialOpener) Open(ctx context.Context, port Port) (Client, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	path := port.OpenPath()
	if path == "" {
		return nil, errors.New("modem: candidate has no AT port")
	}
	baudRate := opener.BaudRate
	if baudRate <= 0 {
		baudRate = 115200
	}
	rawPort, err := serial.Open(path, &serial.Mode{
		BaudRate: baudRate,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	})
	if err != nil {
		return nil, fmt.Errorf("open AT port %s: %w", path, err)
	}
	if err := rawPort.ResetInputBuffer(); err != nil {
		_ = rawPort.Close()
		return nil, fmt.Errorf("reset AT input buffer %s: %w", path, err)
	}
	session, err := NewSession(rawPort, opener.SessionOptions)
	if err != nil {
		_ = rawPort.Close()
		return nil, err
	}
	return session, nil
}
