# lease_hosts

`lease_hosts` is a [CoreDNS](https://github.com/coredns/coredns) plugin that enables DNS resolution for local network hosts by directly monitoring and parsing the `dnsmasq.leases` file.

## Description

When running a decoupled network stack (e.g., **dnsmasq** for DHCP and **CoreDNS** for DNS), CoreDNS typically loses the ability to resolve hostnames assigned via DHCP.

The `lease_hosts` plugin solves this by:

1.  **Direct Parsing**: Reading the `dnsmasq.leases` file to map hostnames to IP addresses.
2.  **Real-time Updates**: Using `inotify` (via `fsnotify`) to watch for file changes. When a new device connects or a lease is updated, the internal memory map is refreshed instantly without restarting CoreDNS.
3.  **Efficiency**: Lookups are performed against an in-memory `std::map`.

## Syntax

```corefile
lease_hosts FILE {
    fallthrough
}
```

  * **FILE**: The absolute path to the `dnsmasq.leases` file (usually `/var/lib/misc/dnsmasq.leases`).
  * **fallthrough**: If enabled, when a query name is not found in the leases file, the plugin will pass the request to the next plugin in the chain (e.g., `forward`). If disabled (default), the plugin returns `NXDOMAIN` for unmapped names.

## Examples

### Basic Usage with .local Rewrite

To resolve hostnames like `my-laptop.local` where `my-laptop` exists in the leases file:

```corefile
. {
    rewrite name suffix .local . answer auto
    lease_hosts /var/lib/misc/dnsmasq.leases {
        fallthrough
    }
    forward . 8.8.8.8
}
```

### Advanced Filtering

If you only want to resolve specific local queries through the leases file:

```corefile
cluster.local {
    lease_hosts /var/lib/misc/dnsmasq.leases
}
```

## Compilation

To compile CoreDNS with this plugin, add the following to `plugin.cfg` after `hosts`:

```text
lease_hosts:github.com/darren/coredns_lease_hosts
```

Then build CoreDNS:

```bash
go get github.com/darren/coredns_lease_hosts
go generate
go build
```

## Internal Lease File Format

The plugin expects the standard `dnsmasq` lease format:
`[expiration_time] [mac_address] [ip_address] [hostname] [client_id]`

Only records with a valid hostname (not `*`) will be loaded into the memory map.

## See Also

See the [dnsmasq documentation](http://www.thekelleys.org.uk/dnsmasq/doc.html) for details on DHCP lease management.