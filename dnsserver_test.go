package dnsserver

import (
	"fmt"
	"net"
	"reflect"
	"testing"

	"github.com/erikh/dnsserver/db"
	"github.com/miekg/dns"
)

// 53 could be in use by a local cache or w/e. 5353 is in use by avahi mDNS on
// my machine. Basically I don't want to have to write a port allocator.
const service = "127.0.0.1:5300"

var server = New("docker")

func init() {
	go func() {
		err := server.Listen(service)
		if err != nil {
			panic(err)
		}
	}()
}

func msgClient(fqdn string, dnsType uint16) (*dns.Msg, error) {
	m := new(dns.Msg)
	m.SetQuestion(fqdn, dnsType)
	return dns.Exchange(m, service)
}

func BenchmarkARecordQueries(b *testing.B) {
	table := map[string]net.IP{
		"test":  net.ParseIP("127.0.0.2"),
		"test2": net.ParseIP("127.0.0.3"),
	}

	// do this in independent parts so both records exist. This tests some
	// collision issues.
	for host, ip := range table {
		server.SetA(host, ip)
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			msg, err := msgClient("test.docker.", dns.TypeA)
			if err != nil {
				b.Log(err)
				continue
			}
			aRecord := msg.Answer[0].(*dns.A).A
			if !aRecord.Equal(table["test"]) {
				b.Fatalf("IP %q does not match registered IP %q", aRecord, table["test"])
			}
		}
	})
}

func TestARecordCRUD(t *testing.T) {
	table := map[string]net.IP{
		"test":  net.ParseIP("127.0.0.2"),
		"test2": net.ParseIP("127.0.0.3"),
	}

	// do this in independent parts so both records exist. This tests some
	// collision issues.
	for host, ip := range table {
		server.SetA(host, ip)
	}

	recs, err := server.ListA()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(recs, table) {
		t.Fatal("tables are not equal")
	}

	// copy/mod check

	recs["test"] = net.ParseIP("5.4.3.2")

	recs2, err := server.ListA()
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(recs, recs2) {
		t.Fatal("listing able to modify inner structs of dns server")
	}

	// general message tests
	for host, ip := range table {
		msg, err := msgClient(fmt.Sprintf("%s.docker.", host), dns.TypeA)
		if err != nil {
			t.Fatal(err)
		}

		if len(msg.Answer) != 1 {
			t.Fatalf("Server did not reply with a valid answer.")
		}

		if msg.Answer[0].Header().Ttl != 1 {
			t.Fatalf("TTL was %d instead of 1", msg.Answer[0].Header().Ttl)
		}

		if msg.Answer[0].Header().Rrtype != dns.TypeA {
			t.Fatalf("Expected A record, got a record of type %d instead.", msg.Answer[0].Header().Rrtype)
		}

		if msg.Answer[0].Header().Name != fmt.Sprintf("%s.docker.", host) {
			// If this fails, we probably need to look at miekg/dns as this should
			// not be possible.
			t.Fatalf("Name does not match query sent, %q was provided", msg.Answer[0].Header().Name)
		}

		aRecord := msg.Answer[0].(*dns.A).A
		if !aRecord.Equal(ip) {
			t.Fatalf("IP %q does not match registered IP %q", aRecord, ip)
		}
	}

	for host := range table {
		server.DeleteA(host)
	}

	for host := range table {
		msg, err := msgClient(fmt.Sprintf("%s.docker.", host), dns.TypeA)

		if err != nil {
			t.Fatal(err)
		}

		if len(msg.Answer) != 0 {
			t.Fatal("Server gave a reply after record has been deleted")
		}
	}
}

func TestSRVRecordCRUD(t *testing.T) {
	table := map[string]*db.SRVRecord{
		"test":  &db.SRVRecord{Port: 80, Host: "test"},
		"test2": &db.SRVRecord{Port: 81, Host: "test2"},
	}

	// do this in independent parts so both records exist. This tests some
	// collision issues.
	for name, srv := range table {
		server.SetSRV(name, "tcp", srv)
	}

	recs, err := server.ListSRV()
	if err != nil {
		t.Fatal(err)
	}

	for host, srv := range table {
		recSRV := recs["_"+host+"._tcp"] // HACK until I get a better API in
		if !srv.Equal(recSRV) {
			t.Fatalf("srv records were not equal for %q", host)
		}
	}

	// copy+mod check

	recs["_test._tcp"] = &db.SRVRecord{Port: 5150, Host: "test-nope"}
	recs2, err := server.ListSRV()
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(recs, recs2) {
		t.Fatal("was able to modify inner struct in dnsserver after list")
	}

	for name, srv := range table {
		msg, err := msgClient(fmt.Sprintf("_%s._tcp.docker.", name), dns.TypeSRV)

		if err != nil {
			t.Fatal(err)
		}

		if len(msg.Answer) != 1 {
			t.Fatalf("Server did not reply with a valid answer.")
		}

		if msg.Answer[0].Header().Ttl != 1 {
			t.Fatalf("TTL was %d instead of 1", msg.Answer[0].Header().Ttl)
		}

		if msg.Answer[0].Header().Rrtype != dns.TypeSRV {
			t.Fatalf("Expected SRV record, got a record of type %d instead.", msg.Answer[0].Header().Rrtype)
		}

		if msg.Answer[0].Header().Name != fmt.Sprintf("_%s._tcp.docker.", name) {
			// If this fails, we probably need to look at miekg/dns as this should
			// not be possible.
			t.Fatalf("Name does not match query sent, %q was provided", msg.Answer[0].Header().Name)
		}

		srvRecord := msg.Answer[0].(*dns.SRV)

		if srvRecord.Priority != 0 || srvRecord.Weight != 0 {
			t.Fatal("Defaults for priority and weight do not equal 0")
		}

		if srvRecord.Port != srv.Port || srvRecord.Target != fmt.Sprintf("%s.docker.", name) {
			t.Fatalf("SRV records are not equivalent: received host %q port %d", srvRecord.Target, srvRecord.Port)
		}
	}

	for name := range table {
		server.DeleteSRV(name, "tcp")
	}

	for name := range table {
		msg, err := msgClient(fmt.Sprintf("_%s._tcp.docker.", name), dns.TypeSRV)

		if err != nil {
			t.Fatal(err)
		}

		if len(msg.Answer) != 0 {
			t.Fatal("Server gave a reply after record has been deleted")
		}
	}
}
