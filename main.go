package main

import (
	"fmt"
	"os"

	"pault.ag/go/config"

	"github.com/mitchellh/goamz/aws"

	"pault.ag/go/dnsync/amazon"
	"pault.ag/go/dnsync/dns"
	"pault.ag/go/dnsync/dnsmasq"
)

type Config struct {
	Zone           string `flag:"zone" description:"AWS Route 53 Zone ID"`
	RootDomainName string `flag:"root-domain-name" description:"Root domain name (like paultag.house)"`
	Leases         string `flag:"leases" description:"Leases file path"`
}

func ohshit(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	conf := Config{Leases: "/var/lib/misc/dnsmasq.leases"}
	flags, err := config.LoadFlags("dnsync", &conf)
	ohshit(err)
	flags.Parse(os.Args[1:])

	if conf.RootDomainName == "" || conf.Zone == "" {
		panic("No root domain or zone configured")
	}

	fd, err := os.Open(conf.Leases)
	ohshit(err)

	auth, err := aws.EnvAuth()
	client := amazon.New(auth, aws.USWest2, conf.Zone)
	awsEntries, err := client.List(conf.RootDomainName)
	ohshit(err)

	leases, err := dnsmasq.Parse(fd)
	ohshit(err)

	hosts := leases.Hosts(conf.RootDomainName)

	change := dns.Change(awsEntries, hosts)
	fmt.Printf("%s\n", change)

	if len(change) == 0 {
		fmt.Printf("Nothing needs to be done!\n")
		os.Exit(0)
	}
	records, err := client.Update(change)
	ohshit(err)

	fmt.Printf("%s\n", records)
}
