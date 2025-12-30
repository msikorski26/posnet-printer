package main

import (
	"bufio"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"golang.org/x/text/encoding/charmap"
)

const (
	STX byte = 0x02
	ETX byte = 0x03
	TAB byte = 0x09
	LF  byte = 0x0A

	crcPrefix = '#'
)

type Encoding int

const (
	EncCP1250 Encoding = iota
	EncISO88592
	EncMazovia
	EncASCII
)

func parseEncoding(s string) (Encoding, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "cp1250", "windows-1250", "win1250":
		return EncCP1250, nil
	case "latin2", "latin-2", "iso-8859-2", "iso8859-2":
		return EncISO88592, nil
	case "mazovia":
		return EncMazovia, nil
	case "ascii":
		return EncASCII, nil
	default:
		return EncCP1250, fmt.Errorf("unknown encoding: %q (use: cp1250|latin2|mazovia|ascii)", s)
	}
}

func encodeText(enc Encoding, s string) ([]byte, error) {
	switch enc {
	case EncASCII:
		for _, r := range s {
			if r > 0x7F {
				return nil, fmt.Errorf("non-ascii rune %q in %q", r, s)
			}
		}
		return []byte(s), nil

	case EncCP1250:
		return charmap.Windows1250.NewEncoder().Bytes([]byte(s))

	case EncISO88592:
		return charmap.ISO8859_2.NewEncoder().Bytes([]byte(s))

	case EncMazovia:
		return encodeMazoviaPL(s), nil

	default:
		return nil, fmt.Errorf("unsupported encoding")
	}
}

func encodeMazoviaPL(s string) []byte {
	m := map[rune]byte{
		'Ą': 0x8F, 'Ć': 0x95, 'Ę': 0x90, 'Ł': 0x9C, 'Ń': 0xA5, 'Ó': 0xA0, 'Ś': 0x98, 'Ź': 0xA3, 'Ż': 0xA1,
		'ą': 0x86, 'ć': 0x8D, 'ę': 0x91, 'ł': 0x92, 'ń': 0xA4, 'ó': 0xA2, 'ś': 0x9E, 'ź': 0xA6, 'ż': 0xA7,
	}
	out := make([]byte, 0, len(s))
	for _, r := range s {
		if r <= 0x7F {
			out = append(out, byte(r))
			continue
		}
		if b, ok := m[r]; ok {
			out = append(out, b)
			continue
		}
		out = append(out, ' ')
	}
	return out
}

func crc16CCITT(data []byte) uint16 {
	var crc uint16 = 0x0000
	for _, b := range data {
		crc ^= uint16(b) << 8
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

type Client struct {
	conn    net.Conn
	r       *bufio.Reader
	enc     Encoding
	logRX   bool
	logTX   bool
	timeout time.Duration
}

func Dial(ctx context.Context, addr string, enc Encoding, timeout time.Duration, logTX, logRX bool) (*Client, error) {
	d := net.Dialer{Timeout: timeout}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	c := &Client{
		conn:    conn,
		r:       bufio.NewReader(conn),
		enc:     enc,
		logRX:   logRX,
		logTX:   logTX,
		timeout: timeout,
	}
	return c, nil
}

func (c *Client) Close() error { return c.conn.Close() }

func MakeFrame(payload []byte) []byte {
	crc := crc16CCITT(payload)
	crcStr := fmt.Sprintf("%04X", crc)

	out := make([]byte, 0, 1+len(payload)+1+4+1)
	out = append(out, STX)
	out = append(out, payload...)
	out = append(out, crcPrefix)
	out = append(out, []byte(crcStr)...)
	out = append(out, ETX)
	return out
}

func (c *Client) Send(payloadASCII string) error {
	payload := []byte(payloadASCII)

	frame := MakeFrame(payload)
	if c.logTX {
		fmt.Println("TX:", sanitizeASCII(payloadASCII))
	}
	_ = c.conn.SetWriteDeadline(time.Now().Add(c.timeout))
	_, err := c.conn.Write(frame)
	return err
}

func (c *Client) SendBytes(payload []byte) error {
	frame := MakeFrame(payload)
	if c.logTX {
		fmt.Println("TX(bytes):", hex.EncodeToString(payload))
	}
	_ = c.conn.SetWriteDeadline(time.Now().Add(c.timeout))
	_, err := c.conn.Write(frame)
	return err
}

func (c *Client) ReadFrame(ctx context.Context) (string, error) {
	if deadline, ok := ctx.Deadline(); ok {
		_ = c.conn.SetReadDeadline(deadline)
	} else {
		_ = c.conn.SetReadDeadline(time.Now().Add(c.timeout))
	}

	for {
		b, err := c.r.ReadByte()
		if err != nil {
			return "", err
		}
		if b == STX {
			break
		}
	}

	var buf []byte
	for {
		ch, err := c.r.ReadByte()
		if err != nil {
			return "", err
		}
		if ch == ETX {
			break
		}
		buf = append(buf, ch)
	}

	if len(buf) < 5 {
		return "", errors.New("frame too short")
	}
	prefixPos := len(buf) - 5
	if buf[prefixPos] != crcPrefix {
		return "", errors.New("CRC prefix not found at expected position")
	}

	payload := buf[:prefixPos]
	crcHex := string(buf[prefixPos+1:])

	gotBytes, err := hex.DecodeString(crcHex)
	if err != nil || len(gotBytes) != 2 {
		return "", errors.New("CRC decode failed")
	}
	got := uint16(gotBytes[0])<<8 | uint16(gotBytes[1])
	want := crc16CCITT(payload)
	if got != want {
		return "", fmt.Errorf("CRC mismatch: got %04X want %04X", got, want)
	}

	s := string(payload)
	if c.logRX {
		fmt.Println("RX:", sanitizeASCII(s))
	}
	return s, nil
}

func sanitizeASCII(s string) string {
	s = strings.ReplaceAll(s, string([]byte{TAB}), "\\t")
	s = strings.ReplaceAll(s, string([]byte{LF}), "\\n")
	return s
}

type SuperForm200 struct {
	c *Client
}

func (c *Client) Form200Start(fh int, al string) (*SuperForm200, error) {
	var payload []byte
	sb := new(strings.Builder)

	fmt.Fprintf(sb, "formstart%cfn200%c", TAB, TAB)

	if fh >= 0 {
		fmt.Fprintf(sb, "fh%d%c", fh, TAB)
	}

	if al != "" {
		alBytes, err := encodeText(c.enc, al)
		if err != nil {
			return nil, err
		}
		base := []byte(sb.String())
		payload = append(payload, base...)
		payload = append(payload, []byte("al")...)
		payload = append(payload, alBytes...)
		payload = append(payload, TAB)
	} else {
		payload = []byte(sb.String())
	}

	if err := c.SendBytes(payload); err != nil {
		return nil, err
	}
	return &SuperForm200{c: c}, nil
}

func (f *SuperForm200) FormattedLine(s1 string, mask string) error {
	s1b, err := encodeText(f.c.enc, s1)
	if err != nil {
		return err
	}

	var payload []byte
	payload = append(payload, []byte("formformattedline")...)
	payload = append(payload, TAB)
	payload = append(payload, []byte("s1")...)
	payload = append(payload, s1b...)
	payload = append(payload, TAB)
	payload = append(payload, []byte("fn200")...)
	payload = append(payload, TAB)

	if mask != "" {
		mb, err := encodeText(f.c.enc, mask)
		if err != nil {
			return err
		}
		payload = append(payload, []byte("ma")...)
		payload = append(payload, mb...)
		payload = append(payload, TAB)
	}

	return f.c.SendBytes(payload)
}

func (f *SuperForm200) TinyLine(s1 string) error {
	s1b, err := encodeText(f.c.enc, s1)
	if err != nil {
		return err
	}

	var payload []byte
	payload = append(payload, []byte("formtinyline")...)
	payload = append(payload, TAB)
	payload = append(payload, []byte("fn200")...)
	payload = append(payload, TAB)
	payload = append(payload, []byte("s1")...)
	payload = append(payload, s1b...)
	payload = append(payload, TAB)

	return f.c.SendBytes(payload)
}

func (f *SuperForm200) Cmd(cm int) error {
	return f.c.Send(fmt.Sprintf("formcmd%cfn200%ccm%d%c", TAB, TAB, cm, TAB))
}

func (f *SuperForm200) End() error {
	return f.c.Send(fmt.Sprintf("formend%cfn200%c", TAB, TAB))
}
