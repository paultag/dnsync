package amazon

import (
	"net"
	"strings"

	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/route53"

	"pault.ag/go/dnsync/dns"
)

type Client struct {
	client *route53.Route53
	auth   aws.Auth
	region aws.Region
	zone   string
}

func New(auth aws.Auth, region aws.Region, zone string) Client {
	return Client{
		auth:   auth,
		region: region,
		zone:   zone,
		client: route53.New(auth, region),
	}
}

func (e Client) List(root string) (dns.Hosts, error) {
	if res, err := e.client.ListResourceRecordSets(e.zone, nil); err == nil {
		hosts := dns.Hosts{}
		for _, el := range res.Records {
			if el.Type != "A" {
				continue
			}
			name := strings.ToLower(el.Name)
			if strings.HasSuffix(root, el.Name) {
				continue
			}
			hosts = append(hosts, dns.Host{
				FQDN: name,
				IP:   net.ParseIP(el.Records[0]),
			})
		}
		return hosts, nil
	} else {
		return nil, err
	}

}

func (e Client) Update(entries []route53.Change) (*route53.ChangeResourceRecordSetsResponse, error) {
	return e.client.ChangeResourceRecordSets(
		e.zone,
		&route53.ChangeResourceRecordSetsRequest{Changes: entries},
	)

}
