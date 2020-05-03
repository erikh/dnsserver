package db

import (
	"net"
	"sync"
)

// Map is a simple in-memory map of DNS entries. These are 1:1 entries, no
// multiple address forms are permitted.
type Map struct {
	aRecords   ARecords     // FQDN -> IP
	srvRecords SRVRecords   // service (e.g., _test._tcp) -> SRV
	aMutex     sync.RWMutex // mutex for A record operations
	srvMutex   sync.RWMutex // mutex for SRV record operations
}

// NewMap makes a new *Map.
func NewMap() *Map {
	return &Map{
		aRecords:   ARecords{},
		srvRecords: SRVRecords{},
	}
}

// Close does nothing.
func (m *Map) Close() error {
	return nil
}

// SetA overwrites or sets the A record for the entry.
func (m *Map) SetA(host string, ip net.IP) error {
	m.aMutex.Lock()
	m.aRecords[host] = ip
	m.aMutex.Unlock()
	return nil
}

// DeleteA deletes an A record for a host. Note that this is not the FQDN, but a hostname.
func (m *Map) DeleteA(host string) error {
	m.aMutex.Lock()
	delete(m.aRecords, host)
	m.aMutex.Unlock()

	return nil
}

// GetA retrieves an A record by FQDN.
func (m *Map) GetA(fqdn string) (net.IP, error) {
	m.aMutex.RLock()
	defer m.aMutex.RUnlock()
	val, ok := m.aRecords[fqdn]
	if !ok {
		return nil, ErrNotFound
	}

	return val, nil
}

// ListA lists all the A records in the database.
func (m *Map) ListA() (ARecords, error) {
	m.aMutex.RLock()
	defer m.aMutex.RUnlock()

	tmp := ARecords{}

	for name, rec := range m.aRecords {
		tmp[name] = rec[:]
	}

	return tmp, nil
}

// SetSRV sets a srv record with service and protocol pointing at a name and port.
func (m *Map) SetSRV(spec string, srv *SRVRecord) error {
	m.srvMutex.Lock()
	m.srvRecords[spec] = srv
	m.srvMutex.Unlock()
	return nil
}

// GetSRV gets a service based on a name
func (m *Map) GetSRV(spec string) (*SRVRecord, error) {
	m.srvMutex.RLock()
	defer m.srvMutex.RUnlock()

	srv, ok := m.srvRecords[spec]
	if !ok {
		return nil, ErrNotFound
	}

	return srv, nil
}

// ListSRV lists all SRV records in the database.
func (m *Map) ListSRV() (SRVRecords, error) {
	tmp := SRVRecords{}

	m.srvMutex.RLock()
	defer m.srvMutex.RUnlock()

	for name, rec := range m.srvRecords {
		t := *rec
		tmp[name] = &t
	}

	return tmp, nil
}

// DeleteSRV deletes a SRV record based on the service and protocol.
func (m *Map) DeleteSRV(spec string) error {
	m.srvMutex.Lock()
	delete(m.srvRecords, spec)
	m.srvMutex.Unlock()

	return nil
}
