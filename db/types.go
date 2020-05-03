package db

import "net"

// ARecords is a typed mapping of A records.
type ARecords map[string]net.IP

// SRVRecords is likewise a collection of SRV records.
type SRVRecords map[string]*SRVRecord

// SRVRecord encapsulates the data segment of a SRV record. Priority and Weight
// are always 0 in our SRV records.
type SRVRecord struct {
	Port uint16
	Host string
}

// Equal tests if the srvrecords are equal.
func (s *SRVRecord) Equal(s2 *SRVRecord) bool {
	return s.Port == s2.Port && s.Host == s2.Host
}
