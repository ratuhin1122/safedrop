package httpx

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// TrustedProxy only accepts forwarding headers from configured network peers.
// Walk right to left, stopping at the first address outside the trusted chain.
func TrustedProxy(prefixes []netip.Prefix) func(http.Handler) http.Handler {
	trusted := append([]netip.Prefix(nil), prefixes...)
	contains := func(ip netip.Addr) bool {
		for _, prefix := range trusted {
			if prefix.Contains(ip.Unmap()) {
				return true
			}
		}
		return false
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			peer, parseErr := netip.ParseAddr(host)
			if err == nil && parseErr == nil && contains(peer) {
				chain := strings.Split(strings.Join(r.Header.Values("X-Forwarded-For"), ","), ",")
				client := peer
				valid := true
				for i := len(chain) - 1; i >= 0 && contains(client); i-- {
					client, err = netip.ParseAddr(strings.TrimSpace(chain[i]))
					if err != nil || client.Zone() != "" {
						valid = false
						break
					}
				}
				if valid {
					r.RemoteAddr = client.Unmap().String()
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
