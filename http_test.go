package ratelimit

import (
	"net/http"
	"testing"
)

func nameForRemoteAddr(n HTTPRemoteNamer, addr string) string {
	var req http.Request
	req.RemoteAddr = addr
	return n.Name(&req)
}

func nameForRemoteIP(n HTTPRemoteNamer, ip string) string {
	var req http.Request
	req.Header = http.Header{}
	req.Header.Set("X-Forwarded-For", ip)
	return n.Name(&req)
}

func TestHTTPRemoteNamer(t *testing.T) {
	namer := HTTPRemoteNamer{34}
	if nameForRemoteAddr(namer, "[2001:0db8:ac10:fe01::]:1234") !=
		"0010000000000001000011011011100010" {
		t.Error("unexpected result")
	}
	if nameForRemoteAddr(namer, "123.76.192.1:1234") != "123.76.192.1" {
		t.Error("unexpected result")
	}
	if nameForRemoteIP(namer, "2001:0db8:ac10:fe01::") !=
		"0010000000000001000011011011100010" {
		t.Error("unexpected result")
	}
	if nameForRemoteIP(namer, "123.76.192.1") != "123.76.192.1" {
		t.Error("unexpected result")
	}

	namer = HTTPRemoteNamer{}
	if nameForRemoteAddr(namer, "[2001:0db8:ac10:fe01::]:1234") !=
		"0010000000000001000011011011100010101100000100001111111000000001" {
		t.Error("unexpected result")
	}
}
