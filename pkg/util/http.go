package util

import (
	"net"
	"net/http"
)

func WriteStatusCodeError(w http.ResponseWriter, code int) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
}

// GetTrustedIP returns the trusted IP address of the client. It checks the
// X-Forwarded-For header first, and falls back to the RemoteAddr field of the
// request if the header is not present. depth is the number of proxies that
// the request has passed through.
func GetTrustedIP(r *http.Request, depth int) string {
	ips := r.Header.Values("X-Forwarded-For")
	if len(ips) == 0 || depth > len(ips) {
		remoteHost, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			remoteHost = r.RemoteAddr
		}
		return remoteHost
	}
	return ips[len(ips)-depth]
}
