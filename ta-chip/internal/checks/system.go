package checks

import (
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"os"
	"time"
)

func GetHostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}

// CheckTimeDate queries pool.ntp.org and compares to local time.
// Returns status (V/Y/X) and a human-readable detail string.
func CheckTimeDate(toleranceSecs int) (status, detail string) {
	delta, err := ntpDelta("pool.ntp.org:123")
	if err != nil {
		return "X", fmt.Sprintf("NTP unreachable: %v", err)
	}

	abs := delta
	if abs < 0 {
		abs = -abs
	}

	switch {
	case abs < float64(toleranceSecs):
		return "V", fmt.Sprintf("drift %.1fs", delta)
	case abs < float64(toleranceSecs*5):
		return "Y", fmt.Sprintf("drift %.1fs (outside %ds tolerance)", delta, toleranceSecs)
	default:
		return "X", fmt.Sprintf("drift %.1fs — clock is off", delta)
	}
}

// ntpDelta returns (localTime - ntpTime) in seconds.
func ntpDelta(addr string) (float64, error) {
	conn, err := net.DialTimeout("udp", addr, 5*time.Second)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(5 * time.Second))

	req := make([]byte, 48)
	req[0] = 0x1B // LI=0, VN=3, Mode=3 (client)
	if _, err := conn.Write(req); err != nil {
		return 0, err
	}

	resp := make([]byte, 48)
	if _, err := conn.Read(resp); err != nil {
		return 0, err
	}

	// Transmit timestamp is at bytes 40-47 (NTP epoch: Jan 1 1900)
	secs := binary.BigEndian.Uint32(resp[40:44])
	frac := binary.BigEndian.Uint32(resp[44:48])

	ntpEpoch := time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)
	ntpTime := ntpEpoch.Add(time.Duration(secs) * time.Second).
		Add(time.Duration(math.Round(float64(frac)/float64(1<<32)*1e9)) * time.Nanosecond)

	delta := time.Since(ntpTime).Seconds()
	return delta, nil
}
