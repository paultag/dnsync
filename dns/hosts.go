package dns

import (
	"net"
	"sort"

	"github.com/mitchellh/goamz/route53"
)

type Host struct {
	FQDN string
	IP   net.IP
}

func (h Host) Equal(o Host) bool {
	return (h.FQDN == o.FQDN && h.IP.Equal(o.IP))
}

func (h Host) Less(o Host) bool {
	return h.FQDN < o.FQDN
}

func (h Host) Change(action string) route53.Change {
	return route53.Change{
		Action: action,
		Record: route53.ResourceRecordSet{
			Name:    h.FQDN,
			Type:    "A",
			TTL:     300,
			Records: []string{h.IP.String()},
		},
	}
}

type Hosts []Host

func (slice Hosts) Len() int {
	return len(slice)
}

func (slice Hosts) Less(i, j int) bool {
	return slice[i].Less(slice[j])
}

func (slice Hosts) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

func Change(oldHosts Hosts, newHosts Hosts) []route53.Change {
	sort.Sort(oldHosts)
	sort.Sort(newHosts)

	toDelete := Hosts{}
	toAdd := Hosts{}

	i := 0
	j := 0

	for i < len(oldHosts) && j < len(newHosts) {
		oldEl := oldHosts[i]
		newEl := newHosts[j]

		if oldEl.FQDN == newEl.FQDN {
			if oldEl.Equal(newEl) {
				i++
				j++
				continue
			}
			toDelete = append(toDelete, oldEl)
			toAdd = append(toAdd, newEl)
			i++
			j++
			continue
		}

		/*
			A B   D E F     oldHosts
			  B C D   F G   newHosts
		*/

		if oldEl.Less(newEl) {
			/* This means we have a oldEl that isn't in the new list */
			toDelete = append(toDelete, oldEl)
			i++
			continue
		}

		/* This means we have a newEl that isn't in the old list */
		toAdd = append(toAdd, newEl)
		j++
	}

	toDelete = append(toDelete, oldHosts[i:]...)
	toAdd = append(toAdd, newHosts[j:]...)

	ret := []route53.Change{}

	for _, el := range toDelete {
		ret = append(ret, el.Change("DELETE"))
	}

	for _, el := range toAdd {
		ret = append(ret, el.Change("CREATE"))
	}

	return ret
}
