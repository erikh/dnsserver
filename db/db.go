package db

import (
	"errors"
	"net"
)

// DB is for pluggable backends. To swap out the DB implementation, implement
// this interface and provide it at construction time to the DNS server struct.
type DB interface {
	SetA(string, net.IP) error
	GetA(string) (net.IP, error)
	DeleteA(string) error
	ListA() (map[string]net.IP, error)
	SetSRV(string, *SRVRecord) error
	GetSRV(string) (*SRVRecord, error)
	DeleteSRV(string) error
	ListSRV() (map[string]*SRVRecord, error)
	Close() error
}

// ErrNotFound is for when the record cannot be located
var ErrNotFound = errors.New("not found")
