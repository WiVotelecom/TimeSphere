// Package ntp provides NTP client functionality for querying time servers.
package ntp

import (
	"fmt"
	"net"
	"time"
)

// NTP Constants
const (
	ntpPort       = 123
	maxStratum    = 15
	ntpLeapAlert  = 3 // Leap indicator value indicating alarm condition
)

// NTP Packet structure (RFC 5905 simplified)
type ntpPacket struct {
	LiVnMode       uint8  // Leap Indicator, Version, Mode
	Stratum        uint8  // Stratum level
	Poll           int8   // Poll interval in seconds
	Precision      int8   // Precision in seconds
	RootDelay      uint32 // Total round trip delay
	RootDispersion uint32 // Max error allowance
	ReferenceID    uint32 // Reference ID
	RefTimeSec     uint32 // Reference time seconds
	RefTimeFrac    uint32 // Reference time fraction
	OrigTimeSec    uint32 // Originate time seconds
	OrigTimeFrac   uint32 // Originate time fraction
	RxTimeSec      uint32 // Receive time seconds
	RxTimeFrac     uint32 // Receive time fraction
	TxTimeSec      uint32 // Transmit time seconds
	TxTimeFrac     uint32 // Transmit time fraction
}

// ServerStatus represents the status of an NTP server query.
type ServerStatus struct {
	Name           string        `json:"name"`
	Address        string        `json:"address"`
	Offset         time.Duration `json:"offset"`          // Time offset from server
	Delay          time.Duration `json:"delay"`           // Round trip delay
	Jitter         time.Duration `json:"jitter"`          // Estimated jitter
	Reachable      bool          `json:"reachable"`       // Server is reachable
	PollInterval   int           `json:"poll_interval"`   // Poll interval in seconds
	Stratum        uint8         `json:"stratum"`         // Server stratum
	LeapIndicator  uint8         `json:"leap_indicator"`  // Leap indicator
	RootDispersion time.Duration `json:"root_dispersion"` // Root dispersion
	Precision      time.Duration `json:"precision"`       // Clock precision
	LastQuery      time.Time     `json:"last_query"`      // Last successful query
	Error          string        `json:"error,omitempty"` // Error message if any
	RTT            time.Duration `json:"rtt"`             // Round trip time
}

// QueryNTP queries an NTP server and returns timing information.
func QueryNTP(address string, port int, timeout time.Duration) (*ServerStatus, error) {
	if port == 0 {
		port = ntpPort
	}

	conn, err := net.DialTimeout("udp", fmt.Sprintf("%s:%d", address, port), timeout)
	if err != nil {
		return &ServerStatus{
			Address:   address,
			Reachable: false,
			Error:     fmt.Sprintf("connection failed: %v", err),
		}, err
	}
	defer conn.Close()

	// Set read/write deadlines
	conn.SetDeadline(time.Now().Add(timeout))

	// Create NTP packet (client mode = 3)
	req := &ntpPacket{
		LiVnMode: 0x1B, // LI=0, VN=3, Mode=3 (client)
	}

	// Set transmit timestamp
	now := time.Now()
	sec, frac := ntpTime(now)
	req.TxTimeSec = sec
	req.TxTimeFrac = frac

	// Send request
	if _, err := conn.Write(packNTP(req)); err != nil {
		return &ServerStatus{
			Address:   address,
			Reachable: false,
			Error:     fmt.Sprintf("write failed: %v", err),
		}, err
	}

	// Read response
	resp := &ntpPacket{}
	buf := make([]byte, 48)
	n, err := conn.Read(buf)
	if err != nil {
		return &ServerStatus{
			Address:   address,
			Reachable: false,
			Error:     fmt.Sprintf("read failed: %v", err),
		}, err
	}

	if n < 48 {
		return &ServerStatus{
			Address:   address,
			Reachable: false,
			Error:     "incomplete response",
		}, fmt.Errorf("incomplete response")
	}

	unpackNTP(buf, resp)

	// Calculate timing
	rxTime := time.Now()
	txTime := ntpToTime(resp.TxTimeSec, resp.TxTimeFrac)
	refTime := ntpToTime(resp.RefTimeSec, resp.RefTimeFrac)

	// Calculate offset and delay using standard NTP formulas
	t1 := now                        // Client transmit time
	t2 := ntpToTime(resp.RxTimeSec, resp.RxTimeFrac) // Server receive time
	t3 := txTime                     // Server transmit time
	t4 := rxTime                     // Client receive time

	offset := ((t2.Sub(t1)) + (t3.Sub(t4))) / 2
	delay := (t4.Sub(t1)) - (t3.Sub(t2))

	// Validate stratum
	if resp.Stratum == 0 || resp.Stratum > maxStratum {
		return &ServerStatus{
			Address:   address,
			Reachable: false,
			Error:     fmt.Sprintf("invalid stratum: %d", resp.Stratum),
		}, fmt.Errorf("invalid stratum")
	}

	// Check leap indicator
	if resp.LiVnMode>>6 == ntpLeapAlert {
		return &ServerStatus{
			Address:   address,
			Reachable: false,
			Error:     "leap indicator alarm",
		}, fmt.Errorf("leap indicator alarm")
	}

	status := &ServerStatus{
		Address:        address,
		Offset:         offset,
		Delay:          delay,
		Jitter:         estimateJitter(offset),
		Reachable:      true,
		PollInterval:   int(resp.Poll),
		Stratum:        resp.Stratum,
		LeapIndicator:  resp.LiVnMode >> 6,
		RootDispersion: microsecondsToDuration(uint32(resp.RootDispersion)),
		Precision:      powerOfTwoToDuration(resp.Precision),
		LastQuery:      time.Now(),
		RTT:            delay,
	}

	return status, nil
}

// ntpTime converts a Go time to NTP timestamp format.
func ntpTime(t time.Time) (uint32, uint32) {
	const ntpEpochOffset = 2208988800 // Seconds between 1900 and 1970
	sec := uint32(t.Unix() + ntpEpochOffset)
	frac := uint32(t.Nanosecond()) * 4294967296 / 1000000000
	return sec, frac
}

// ntpToTime converts NTP timestamp to Go time.
func ntpToTime(sec, frac uint32) time.Time {
	const ntpEpochOffset = 2208988800
	unixSec := int64(sec - ntpEpochOffset)
	nsec := int64(frac) * 1000000000 / 4294967296
	return time.Unix(unixSec, nsec)
}

// packNTP serializes an NTP packet to bytes.
func packNTP(p *ntpPacket) []byte {
	buf := make([]byte, 48)
	buf[0] = p.LiVnMode
	buf[1] = p.Stratum
	buf[2] = byte(p.Poll)
	buf[3] = byte(p.Precision)
	buf[4] = byte(p.RootDelay >> 24)
	buf[5] = byte(p.RootDelay >> 16)
	buf[6] = byte(p.RootDelay >> 8)
	buf[7] = byte(p.RootDelay)
	buf[8] = byte(p.RootDispersion >> 24)
	buf[9] = byte(p.RootDispersion >> 16)
	buf[10] = byte(p.RootDispersion >> 8)
	buf[11] = byte(p.RootDispersion)
	buf[12] = byte(p.ReferenceID >> 24)
	buf[13] = byte(p.ReferenceID >> 16)
	buf[14] = byte(p.ReferenceID >> 8)
	buf[15] = byte(p.ReferenceID)
	buf[16] = byte(p.RefTimeSec >> 24)
	buf[17] = byte(p.RefTimeSec >> 16)
	buf[18] = byte(p.RefTimeSec >> 8)
	buf[19] = byte(p.RefTimeSec)
	buf[20] = byte(p.RefTimeFrac >> 24)
	buf[21] = byte(p.RefTimeFrac >> 16)
	buf[22] = byte(p.RefTimeFrac >> 8)
	buf[23] = byte(p.RefTimeFrac)
	buf[24] = byte(p.OrigTimeSec >> 24)
	buf[25] = byte(p.OrigTimeSec >> 16)
	buf[26] = byte(p.OrigTimeSec >> 8)
	buf[27] = byte(p.OrigTimeSec)
	buf[28] = byte(p.OrigTimeFrac >> 24)
	buf[29] = byte(p.OrigTimeFrac >> 16)
	buf[30] = byte(p.OrigTimeFrac >> 8)
	buf[31] = byte(p.OrigTimeFrac)
	buf[32] = byte(p.RxTimeSec >> 24)
	buf[33] = byte(p.RxTimeSec >> 16)
	buf[34] = byte(p.RxTimeSec >> 8)
	buf[35] = byte(p.RxTimeSec)
	buf[36] = byte(p.RxTimeFrac >> 24)
	buf[37] = byte(p.RxTimeFrac >> 16)
	buf[38] = byte(p.RxTimeFrac >> 8)
	buf[39] = byte(p.RxTimeFrac)
	buf[40] = byte(p.TxTimeSec >> 24)
	buf[41] = byte(p.TxTimeSec >> 16)
	buf[42] = byte(p.TxTimeSec >> 8)
	buf[43] = byte(p.TxTimeSec)
	buf[44] = byte(p.TxTimeFrac >> 24)
	buf[45] = byte(p.TxTimeFrac >> 16)
	buf[46] = byte(p.TxTimeFrac >> 8)
	buf[47] = byte(p.TxTimeFrac)
	return buf
}

// unpackNTP deserializes bytes to an NTP packet.
func unpackNTP(buf []byte, p *ntpPacket) {
	p.LiVnMode = buf[0]
	p.Stratum = buf[1]
	p.Poll = int8(buf[2])
	p.Precision = int8(buf[3])
	p.RootDelay = uint32(buf[4])<<24 | uint32(buf[5])<<16 | uint32(buf[6])<<8 | uint32(buf[7])
	p.RootDispersion = uint32(buf[8])<<24 | uint32(buf[9])<<16 | uint32(buf[10])<<8 | uint32(buf[11])
	p.ReferenceID = uint32(buf[12])<<24 | uint32(buf[13])<<16 | uint32(buf[14])<<8 | uint32(buf[15])
	p.RefTimeSec = uint32(buf[16])<<24 | uint32(buf[17])<<16 | uint32(buf[18])<<8 | uint32(buf[19])
	p.RefTimeFrac = uint32(buf[20])<<24 | uint32(buf[21])<<16 | uint32(buf[22])<<8 | uint32(buf[23])
	p.OrigTimeSec = uint32(buf[24])<<24 | uint32(buf[25])<<16 | uint32(buf[26])<<8 | uint32(buf[27])
	p.OrigTimeFrac = uint32(buf[28])<<24 | uint32(buf[29])<<16 | uint32(buf[30])<<8 | uint32(buf[31])
	p.RxTimeSec = uint32(buf[32])<<24 | uint32(buf[33])<<16 | uint32(buf[34])<<8 | uint32(buf[35])
	p.RxTimeFrac = uint32(buf[36])<<24 | uint32(buf[37])<<16 | uint32(buf[38])<<8 | uint32(buf[39])
	p.TxTimeSec = uint32(buf[40])<<24 | uint32(buf[41])<<16 | uint32(buf[42])<<8 | uint32(buf[43])
	p.TxTimeFrac = uint32(buf[44])<<24 | uint32(buf[45])<<16 | uint32(buf[46])<<8 | uint32(buf[47])
}

// microsecondsToDuration converts microseconds to time.Duration.
func microsecondsToDuration(us uint32) time.Duration {
	return time.Duration(us) * time.Microsecond
}

// powerOfTwoToDuration converts a power-of-two seconds value to duration.
func powerOfTwoToDuration(exp int8) time.Duration {
	if exp >= 0 {
		return time.Second << exp
	}
	return time.Second >> (-exp)
}

// estimateJitter provides a simple jitter estimate based on offset variance.
func estimateJitter(offset time.Duration) time.Duration {
	// Simple estimation - in production this would use statistical analysis
	if offset < 0 {
		offset = -offset
	}
	return offset / 10
}
