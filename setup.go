package lease_hosts

import (
	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
)

func init() { plugin.Register("lease_hosts", setup) }

func setup(c *caddy.Controller) error {
	lh := &LeaseHosts{Hosts: make(map[string]string)}
	for c.Next() {
		if !c.Args(&lh.FilePath) {
			return plugin.Error("lease_hosts", c.ArgErr())
		}

		for c.NextBlock() {
			switch c.Val() {
			case "fallthrough":
				lh.Fallthrough = true
			default:
				return plugin.Error("lease_hosts", c.Errf("unknown property '%s'", c.Val()))
			}
		}
	}

	lh.ParseLeases()

	go lh.watchLeases()

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		lh.Next = next
		return lh
	})
	return nil
}
