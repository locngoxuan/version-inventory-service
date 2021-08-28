package main

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"net"
	"os"
	"sync/atomic"
	"time"
)

// ErrInvalidHex indicates that a hex string cannot be converted to an ObjectID.
var ErrInvalidHex = errors.New("the provided hex string is not a valid ObjectID")

// ObjectID is the BSON ObjectID type.
type ObjectID [12]byte

// NilObjectID is the zero value for ObjectID.
var NilObjectID ObjectID

var objectIDCounter = readRandomUint32()
var machineUnique = getMachineIdentifier()

// NewObjectID generates a new ObjectID.
func NewObjectID() ObjectID {
	return NewObjectIDFromTimestamp(time.Now())
}

// NewObjectIDFromTimestamp generates a new ObjectID based on the given time.
func NewObjectIDFromTimestamp(timestamp time.Time) ObjectID {
	var b [12]byte

	binary.BigEndian.PutUint32(b[0:4], uint32(timestamp.Unix()))
	copy(b[4:9], machineUnique[:])
	putUint24(b[9:12], atomic.AddUint32(&objectIDCounter, 1))

	return b
}

// Timestamp extracts the time part of the ObjectId.
func (id ObjectID) Timestamp() time.Time {
	unixSecs := binary.BigEndian.Uint32(id[0:4])
	return time.Unix(int64(unixSecs), 0).UTC()
}

// Hex returns the hex encoding of the ObjectID as a string.
func (id ObjectID) Hex() string {
	return hex.EncodeToString(id[:])
}

func (id ObjectID) String() string {
	return fmt.Sprintf("ObjectID(%q)", id.Hex())
}

// IsZero returns true if id is the empty ObjectID.
func (id ObjectID) IsZero() bool {
	return bytes.Equal(id[:], NilObjectID[:])
}

// ObjectIDFromHex creates a new ObjectID from a hex string. It returns an error if the hex string is not a
// valid ObjectID.
func ObjectIDFromHex(s string) (ObjectID, error) {
	if len(s) != 24 {
		return NilObjectID, ErrInvalidHex
	}

	b, err := hex.DecodeString(s)
	if err != nil {
		return NilObjectID, err
	}

	var oid [12]byte
	copy(oid[:], b[:])

	return oid, nil
}

// IsValidObjectID returns true if the provided hex string represents a valid ObjectID and false if not.
func IsValidObjectID(s string) bool {
	_, err := ObjectIDFromHex(s)
	return err == nil
}

func getMachineIdentifier() [5]byte {
	var b [5]byte
	mac, err := macAddressBytes()
	if err != nil {
		fmt.Println(fmt.Errorf(`can not get mac address %v`, err))
		os.Exit(1)
	}
	pid := processUniqueBytes()
	copy(b[0:], mac[:])
	copy(b[3:], pid[:])
	return b
}

func macAddressBytes() ([3]byte, error) {
	var b [3]byte
	ifas, err := net.Interfaces()
	if err != nil {
		return b, err
	}
	var as []byte
	for _, ifa := range ifas {
		if ifa.HardwareAddr != nil {
			as = append(as, ifa.HardwareAddr...)
		}
	}
	h := fnv.New32a()
	h.Write(as)
	v := h.Sum32()
	putUint24(b[:], v)
	return b, nil
}

func processUniqueBytes() [2]byte {
	var b [2]byte
	pid := os.Getpid()
	putUint16(b[:], uint32(pid))
	return b
}

func readRandomUint32() uint32 {
	var b [4]byte
	_, err := io.ReadFull(rand.Reader, b[:])
	if err != nil {
		panic(fmt.Errorf("cannot initialize objectid package with crypto.rand.Reader: %v", err))
	}

	return (uint32(b[0]) << 0) | (uint32(b[1]) << 8) | (uint32(b[2]) << 16) | (uint32(b[3]) << 24)
}

func putUint24(b []byte, v uint32) {
	b[0] = byte(v >> 16)
	b[1] = byte(v >> 8)
	b[2] = byte(v)
}

func putUint16(b []byte, v uint32) {
	b[0] = byte(v >> 8)
	b[1] = byte(v)
}
