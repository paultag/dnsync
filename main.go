package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"

	"pault.ag/go/config"

	"github.com/mitchellh/goamz/aws"
	"golang.org/x/exp/inotify"

	"pault.ag/go/dnsync/amazon"
	"pault.ag/go/dnsync/dns"
	"pault.ag/go/dnsync/dnsmasq"
)

type Config struct {
	Zone            string `flag:"zone" description:"AWS Route 53 Zone ID"`
	RootDomainName  string `flag:"root-domain-name" description:"Root domain name (like paultag.house)"`
	Leases          string `flag:"leases" description:"Leases file path"`
	PublicInterface string `flag:"public-interface" description:"public interface (like eth0)"`
}

func ohshit(err error) {
	if err != nil {
		panic(err)
	}
}

func PublicIP(conf Config) *dns.Host {
	ifaces, err := net.Interfaces()
	ohshit(err)
	for _, i := range ifaces {
		if i.Name != conf.PublicInterface {
			continue
		}
		addrs, err := i.Addrs()
		ohshit(err)
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			return &dns.Host{
				FQDN: conf.RootDomainName,
				IP:   ip,
			}
		}
	}
	return nil
}

func Sync(conf Config, client amazon.Client) {
	publicIp := PublicIP(conf)
	if publicIp == nil {
		panic(fmt.Errorf("No public IP found; interface name wrong?"))
	}

	Update(*publicIp, conf, client)

	watcher, err := inotify.NewWatcher()
	ohshit(err)
	ohshit(watcher.Watch(filepath.Dir(conf.Leases)))
	for {
		select {
		case ev := <-watcher.Event:
			fmt.Printf("%s %s %s\n", ev.Mask, ev.Name, ev)
			if ((ev.Mask ^ inotify.IN_MODIFY) != 0) || ev.Name != conf.Leases {
				continue
			}
			Update(*publicIp, conf, client)
		}
	}
}

func Update(publicIp dns.Host, conf Config, client amazon.Client) {
	awsEntries, err := client.List(conf.RootDomainName)
	ohshit(err)

	fd, err := os.Open(conf.Leases)
	ohshit(err)
	leases, err := dnsmasq.Parse(fd)
	ohshit(err)

	hosts := leases.Hosts(conf.RootDomainName)
	hosts = append(hosts, publicIp)

	change := dns.Change(awsEntries, hosts)
	fmt.Printf("%s\n", change)

	if len(change) == 0 {
		fmt.Printf("Nothing needs to be done!\n")
		return
	}
	records, err := client.Update(change)
	ohshit(err)
	fmt.Printf("%s\n", records)
}

func main() {
	conf := Config{
		Leases:          "/var/lib/misc/dnsmasq.leases",
		PublicInterface: "eth0",
	}

	flags, err := config.LoadFlags("dnsync", &conf)
	ohshit(err)
	flags.Parse(os.Args[1:])

	if conf.RootDomainName == "" || conf.Zone == "" {
		panic("No root domain or zone configured")
	}

	auth, err := aws.EnvAuth()
	client := amazon.New(auth, aws.USWest2, conf.Zone)

	Sync(conf, client)
}
