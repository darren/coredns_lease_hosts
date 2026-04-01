package lease_hosts

import (
	"bufio"
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/request"
	"github.com/fsnotify/fsnotify"
	"github.com/miekg/dns"
)

type LeaseHosts struct {
	Next        plugin.Handler
	FilePath    string
	Mux         sync.RWMutex
	Hosts       map[string]string // hostname -> IP
	Fallthrough bool
}

func (lh *LeaseHosts) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}
	qname := strings.ToLower(strings.TrimSuffix(state.Name(), "."))

	lh.Mux.RLock()
	ip, ok := lh.Hosts[qname]
	lh.Mux.RUnlock()


	if ok && state.QType() == dns.TypeA {
		m := new(dns.Msg)
		m.SetReply(r)
		m.Authoritative = true
		m.Answer = []dns.RR{
			&dns.A{
				Hdr: dns.RR_Header{Name: state.Name(), Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60},
				A:   net.ParseIP(ip),
			},
		}
		w.WriteMsg(m)
		return dns.RcodeSuccess, nil
	}


	if lh.Fallthrough {
		return plugin.NextOrFailure(lh.Name(), lh.Next, ctx, w, r)
	}

	// 如果不 fallthrough，通常返回 NXDOMAIN 表示查无此域名
	m := new(dns.Msg)
	m.SetReply(r)
	m.Rcode = dns.RcodeNameError
	w.WriteMsg(m)
	return dns.RcodeNameError, nil
}

func (lh *LeaseHosts) Name() string { return "lease_hosts" }

func (lh *LeaseHosts) ParseLeases() {
	file, _ := os.Open(lh.FilePath)
	defer file.Close()
	newHosts := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 4 && fields[3] != "*" {
			// fields[2] 是 IP, fields[3] 是主机名
			newHosts[strings.ToLower(fields[3])] = fields[2]
		}
	}
	lh.Mux.Lock()
	lh.Hosts = newHosts
	lh.Mux.Unlock()
}

func (lh *LeaseHosts) watchLeases() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Errorf("Failed to create watcher: %v", err)
		return
	}
	defer watcher.Close()

	dir := filepath.Dir(lh.FilePath)
	err = watcher.Add(dir)
	if err != nil {
		log.Errorf("Failed to watch directory: %v", err)
		return
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Name == lh.FilePath && (event.Has(fsnotify.Write) || event.Has(fsnotify.Create)) {
				log.Infof("Lease file changed, reloading...")
				lh.ParseLeases()
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Errorf("Watcher error: %v", err)
		}
	}
}
