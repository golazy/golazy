// Copyright 2017 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in LICENSE.gorilla.

package lazywebsocket

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"testing"
)

var preparedMessageTests = []struct {
	messageType            int
	isServer               bool
	enableWriteCompression bool
	compressionLevel       int
}{
	// Server
	{TextMessage, true, false, flate.BestSpeed},
	{TextMessage, true, true, flate.BestSpeed},
	{TextMessage, true, true, flate.BestCompression},
	{PingMessage, true, false, flate.BestSpeed},
	{PingMessage, true, true, flate.BestSpeed},

	// Client
	{TextMessage, false, false, flate.BestSpeed},
	{TextMessage, false, true, flate.BestSpeed},
	{TextMessage, false, true, flate.BestCompression},
	{PingMessage, false, false, flate.BestSpeed},
	{PingMessage, false, true, flate.BestSpeed},
}

func TestPreparedMessage(t *testing.T) {
	for _, tt := range preparedMessageTests {
		var data = []byte("this is a test")
		var buf bytes.Buffer
		c := newTestConn(nil, &buf, tt.isServer)
		if tt.enableWriteCompression {
			c.newCompressionWriter = compressNoContextTakeover
		}
		c.SetCompressionLevel(tt.compressionLevel)

		if err := c.WriteMessage(tt.messageType, data); err != nil {
			t.Fatal(err)
		}
		want := buf.String()

		pm, err := NewPreparedMessage(tt.messageType, data)
		if err != nil {
			t.Fatal(err)
		}

		// Scribble on data to ensure that NewPreparedMessage takes a snapshot.
		copy(data, "hello world")

		buf.Reset()
		if err := c.WritePreparedMessage(pm); err != nil {
			t.Fatal(err)
		}
		got := buf.String()

		if tt.isServer {
			if got != want {
				t.Errorf("write message != prepared message for %+v", tt)
			}
			continue
		}

		wantOpCode, wantPayload := parsePreparedTestFrame(t, want)
		gotOpCode, gotPayload := parsePreparedTestFrame(t, got)
		if gotOpCode != wantOpCode || !bytes.Equal(gotPayload, wantPayload) {
			t.Errorf("write message != prepared message for %+v", tt)
		}
	}
}

func parsePreparedTestFrame(t *testing.T, frame string) (byte, []byte) {
	t.Helper()

	data := []byte(frame)
	if len(data) < 2 {
		t.Fatalf("frame too short: %d", len(data))
	}

	opCode := data[0]
	length := int(data[1] & 0x7f)
	offset := 2
	switch length {
	case 126:
		if len(data) < offset+2 {
			t.Fatalf("frame missing uint16 length")
		}
		length = int(binary.BigEndian.Uint16(data[offset : offset+2]))
		offset += 2
	case 127:
		if len(data) < offset+8 {
			t.Fatalf("frame missing uint64 length")
		}
		length64 := binary.BigEndian.Uint64(data[offset : offset+8])
		if int64(length64) < 0 {
			t.Fatalf("frame length too large: %d", length64)
		}
		length = int(length64)
		offset += 8
	}

	masked := data[1]&maskBit != 0
	var key [4]byte
	if masked {
		if len(data) < offset+len(key) {
			t.Fatalf("frame missing mask key")
		}
		copy(key[:], data[offset:offset+len(key)])
		offset += len(key)
	}

	if len(data) != offset+length {
		t.Fatalf("frame length mismatch: got %d bytes, header says %d payload bytes", len(data)-offset, length)
	}
	payload := append([]byte(nil), data[offset:]...)
	if masked {
		maskBytes(key, 0, payload)
	}
	return opCode, payload
}
