package ratelimit

import (
	"net"
	"net/http"
	"strings"
)

const DefaultIPv6Bits = 64

// HTTPRemoteNamer assigns unique IDs to the remote hosts of http requests.
type HTTPRemoteNamer struct {
	// IPv6Bits indicates how many bits of IPv6 addresses should be counted towards the remote
	// address. If this is 0, the default value DefaultIPv6Bits will be used.
	IPv6Bits int
}

// Name generates a unique ID for the source of the HTTP request.
func (h HTTPRemoteNamer) Name(r *http.Request) string {
	ipStr := rawIPFromRequest(r)
	parsed := net.ParseIP(ipStr)
	if parsed == nil || parsed.To4() != nil {
		return ipStr
	} else {
		return h.encodeIPv6Binary(parsed)
	}
}

func (h HTTPRemoteNamer) encodeIPv6Binary(address []byte) string {
	bitCount := h.IPv6Bits
	if bitCount == 0 {
		bitCount = DefaultIPv6Bits
	}
	res := make([]byte, bitCount)
	for bitIndex := 0; bitIndex < bitCount; bitIndex++ {
		byteIndex := bitIndex / 8
		bitShift := uint(7 - (bitIndex % 8))
		if address[byteIndex]&(1<<bitShift) == 0 {
			res[bitIndex] = '0'
		} else {
			res[bitIndex] = '1'
		}
	}
	return string(res)
}

func rawIPFromRemoteAddr(addr string) string {
	if !strings.HasPrefix(addr, "[") {
		// The address is "IPv4Address:port"
		return strings.Split(addr, ":")[0]
	}

	// The address is "[IPv6Address]:port".
	ipv6Addr := strings.Split(addr, "]")[0]
	if len(ipv6Addr) < 1 {
		panic("invalid remote address: " + addr)
	}
	return ipv6Addr[1:]
}

func rawIPFromRequest(r *http.Request) string {
	if forwardHeader := r.Header.Get("X-Forwarded-For"); forwardHeader != "" {
		return strings.Split(forwardHeader, ", ")[0]
	}
	return rawIPFromRemoteAddr(r.RemoteAddr)
}
