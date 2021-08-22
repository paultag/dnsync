package dnsmasq

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"pault.ag/go/dnsync/dns"
)

type Lease struct {
	Expiry   time.Time
	MAC      net.HardwareAddr
	IP       net.IP
	Hostname string
	ClientID string
}

func (l Lease) Host(domain string) dns.Host {
	return dns.Host{
		FQDN: strings.ToLower(fmt.Sprintf("%s.%s", l.Hostname, domain)),
		IP:   l.IP,
	}
}

func (l Lease) MACHost(domain string) dns.Host {
	return dns.Host{
		FQDN: strings.ToLower(fmt.Sprintf("%s.by-mac.%s", strings.Replace(l.MAC.String(), ":", "-", -1), domain)),
		IP:   l.IP,
	}
}

type Leases []Lease

func (ls Leases) InCIDR(network *net.IPNet) Leases {
	ret := Leases{}
	for _, l := range ls {
		if !network.Contains(l.IP) {
			continue
		}
		ret = append(ret, l)
	}
	return ret
}

func (l Leases) Hosts(domain string) dns.Hosts {
	ret := dns.Hosts{}
	for _, el := range l {
		if el.Hostname == "" {
			continue
		}
		ret = append(ret, el.Host(domain))
		ret = append(ret, el.MACHost(domain))
	}
	return ret
}

func ParseLine(line string) (*Lease, error) {
	els := strings.Split(line, " ")
	if len(els) != 5 {
		return nil, fmt.Errorf("Not enough entries in the line")
	}

	when, err := strconv.Atoi(els[0])
	if err != nil {
		return nil, err
	}

	ip := net.ParseIP(els[2])
	mac, err := net.ParseMAC(els[1])
	if err != nil {
		return nil, err
	}

	lease := Lease{
		Expiry: time.Unix(int64(when), 0),
		MAC:    mac,
		IP:     ip,
	}

	if els[3] != "*" {
		lease.Hostname = els[3]
	}
	if els[4] != "*" {
		lease.ClientID = els[4]
	}

	return &lease, nil
}

func Parse(reader io.Reader) (Leases, error) {
	br := bufio.NewReader(reader)
	ret := Leases{}

	for {
		str, err := br.ReadString('\n')
		if err != io.EOF && err != nil {
			return nil, err
		}

		if str == "" {
			break
		}

		lease, err := ParseLine(str)
		if err != nil {
			return nil, err
		}
		ret = append(ret, *lease)

		if err == io.EOF {
			break
		}
	}

	return ret, nil
}
