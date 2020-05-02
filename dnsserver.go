package dnsserver

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/erikh/dnsserver/db"
	"github.com/miekg/dns"
)

// Server is the struct which describes the DNS server.
type Server struct {
	domain      string // using the constructor, this will always end in a '.', making it a FQDN.
	db          db.DB
	server      *dns.Server
	configMutex sync.Mutex // mutex for server configuration operations
	listenIP    net.IP
	listenPort  uint
}

// New creates a new DNS server. Domain is an unqualified domain that will be used
// as the TLD.
func New(domain string) *Server {
	return NewWithDB(domain, db.NewMap())
}

// NewWithDB allows you to provide a custom DB implementation during
// construction.
func NewWithDB(domain string, db db.DB) *Server {
	return &Server{
		domain: domain + ".",
		db:     db,
	}
}

// Listening returns the ip:port of the listener.
func (ds *Server) Listening() (net.IP, uint) {
	ds.configMutex.Lock()
	defer ds.configMutex.Unlock()
	return ds.listenIP, ds.listenPort
}

// Listen for DNS requests. listenSpec is a dotted-quad + port, e.g.,
// 127.0.0.1:53. This function blocks and only returns when the DNS service is
// no longer functioning.
func (ds *Server) Listen(listenSpec string) error {
	ds.configMutex.Lock()
	var lc net.ListenConfig
	conn, err := lc.ListenPacket(context.Background(), "udp", listenSpec)
	if err != nil {
		ds.configMutex.Unlock()
		return err
	}
	ds.server = &dns.Server{PacketConn: conn, Addr: listenSpec, Net: "udp", Handler: ds}
	u := conn.LocalAddr().(*net.UDPAddr)
	ds.listenIP, ds.listenPort = u.IP, uint(u.Port)
	ds.configMutex.Unlock()
	return ds.server.ActivateAndServe()
}

// Close closes the DNS server. If it is not started, nil is returned.
func (ds *Server) Close() error {
	ds.configMutex.Lock()
	defer ds.configMutex.Unlock()

	if ds.server != nil {
		return ds.server.Shutdown()
	}

	return nil
}

// Convenience function to ensure the fqdn is well-formed, and keeps the
// set/delete interface easy.
func (ds *Server) qualifyHost(host string) string {
	return host + "." + ds.domain
}

// Convenience function to ensure that SRV names are well-formed.
func (ds *Server) qualifySrv(service, protocol string) string {
	return fmt.Sprintf("_%s._%s.%s", service, protocol, ds.domain)
}

// rewrites supplied host entries to use the domain this dns server manages.
// Returns and modifies the pointer.
func (ds *Server) qualifySrvHost(srv *db.SRVRecord) *db.SRVRecord {
	srv.Host = ds.qualifyHost(srv.Host)
	return srv
}

// GetA receives a FQDN; looks up and supplies the A record.
func (ds *Server) GetA(fqdn string) []*dns.A {
	val, err := ds.db.GetA(fqdn)
	if err != nil {
		if err != db.ErrNotFound {
			fmt.Println(err)
		}
		return nil
	}

	return []*dns.A{&dns.A{
		Hdr: dns.RR_Header{
			Name:   fqdn,
			Rrtype: dns.TypeA,
			Class:  dns.ClassINET,
			Ttl:    1,
		},
		A: val,
	}}
}

// SetA sets a host to an IP. Note that this is not the FQDN, but a hostname.
func (ds *Server) SetA(host string, ip net.IP) {
	ds.db.SetA(ds.qualifyHost(host), ip)
}

// DeleteA deletes a host. Note that this is not the FQDN, but a hostname.
func (ds *Server) DeleteA(host string) {
	ds.db.DeleteA(ds.qualifyHost(host))
}

// GetSRV given a service spec, looks up and returns an array of *dns.SRV objects.
// These must be massaged into the []dns.RR after the fact.
func (ds *Server) GetSRV(spec string) []*dns.SRV {
	srv, err := ds.db.GetSRV(spec)
	if err != nil {
		if err != db.ErrNotFound {
			fmt.Println(err)
		}
		return nil
	}

	srvRecord := &dns.SRV{
		Hdr: dns.RR_Header{
			Name:   spec,
			Rrtype: dns.TypeSRV,
			Class:  dns.ClassINET,
			// 0 TTL results in UB for DNS resolvers and generally causes problems.
			Ttl: 1,
		},
		Priority: 0,
		Weight:   0,
		Port:     srv.Port,
		Target:   srv.Host,
	}

	return []*dns.SRV{srvRecord}
}

// SetSRV sets a SRV with a service and protocol. See SRVRecord for more information
// on what that requires.
func (ds *Server) SetSRV(service, protocol string, srv *db.SRVRecord) error {
	return ds.db.SetSRV(ds.qualifySrv(service, protocol), ds.qualifySrvHost(srv))
}

// DeleteSRV deletes a SRV record based on the service and protocol.
func (ds *Server) DeleteSRV(service, protocol string) {
	ds.db.DeleteSRV(ds.qualifySrv(service, protocol))
}

// ServeDNS is the main callback for miekg/dns. Collects information about the
// query, constructs a response, and returns it to the connector.
func (ds *Server) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	m := &dns.Msg{}
	m.SetReply(r)

	answers := []dns.RR{}

	for _, question := range r.Question {
		// nil records == not found
		switch question.Qtype {
		case dns.TypeA:
			for _, record := range ds.GetA(question.Name) {
				answers = append(answers, record)
			}
		case dns.TypeSRV:
			for _, record := range ds.GetSRV(question.Name) {
				answers = append(answers, record)
			}
		}
	}

	// If we have no answers, that means we found nothing or didn't get a query
	// we can reply to. Reply with no answers so we ensure the query moves on to
	// the next server.
	if len(answers) == 0 {
		m.SetRcode(r, dns.RcodeNameError)
		w.WriteMsg(m)
		return
	}

	// Without these the glibc resolver gets very angry.
	m.Authoritative = true
	m.RecursionAvailable = false
	m.Answer = answers

	m.SetRcode(r, dns.RcodeSuccess)
	if err := w.WriteMsg(m); err != nil {
		fmt.Println(err)
	}
}
