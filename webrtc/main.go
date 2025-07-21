//go:build !js
// +build !js

package main

import (
	"flag"
	"log"

	"github.com/GRVYDEV/itg-webrtc/server"
)

func main() {
	// Command line flags for configuring the server
	addr := flag.String("addr", "localhost", "http service address")
	ip := flag.String("ip", "none", "IP address for webrtc")
	wsPort := flag.Int("ws-port", 8080, "Port for websocket")
	rtpPort := flag.Int("rtp-port", 65535, "Port for RTP")
	ports := flag.String("ports", "20000-20500", "Port range for webrtc")
	iceSrv := flag.String("ice-servers", "none", "Comma separated list of ICE / STUN servers (optional)")
	sslCert := flag.String("ssl-cert", "", "Ssl cert for websocket (optional)")
	sslKey := flag.String("ssl-key", "", "Ssl key for websocket (optional)")

	flag.Parse()
	log.SetFlags(0)

	cfg := server.Config{
		Addr:       *addr,
		IP:         *ip,
		WSPort:     *wsPort,
		RTPPort:    *rtpPort,
		Ports:      *ports,
		ICEServers: *iceSrv,
		SSLCert:    *sslCert,
		SSLKey:     *sslKey,
	}

	s, err := server.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	if err := s.Start(); err != nil {
		log.Fatal(err)
	}
}
