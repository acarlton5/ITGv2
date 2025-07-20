// Package server implements the WebRTC signaling and RTP handling logic.
package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pion/interceptor"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"

	"github.com/GRVYDEV/lightspeed-webrtc/ws"
)

// Config stores runtime options for the WebRTC server.
type Config struct {
	Addr       string
	IP         string
	WSPort     int
	RTPPort    int
	Ports      string
	ICEServers string
	SSLCert    string
	SSLKey     string
}

// Server wraps all state required for serving WebRTC streams.
type Server struct {
	cfg        Config
	upgrader   websocket.Upgrader
	videoTrack *webrtc.TrackLocalStaticRTP
	audioTrack *webrtc.TrackLocalStaticRTP
	hub        *ws.Hub
}

// New creates a new server instance using the provided configuration.
func New(cfg Config) (*Server, error) {
	// Create RTP tracks that are shared by all peers.
	v, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: "video/H264"}, "video", "pion")
	if err != nil {
		return nil, err
	}
	a, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: "audio/opus"}, "audio", "pion")
	if err != nil {
		return nil, err
	}

	return &Server{
		cfg:        cfg,
		upgrader:   websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		videoTrack: v,
		audioTrack: a,
		hub:        ws.NewHub(),
	}, nil
}

// Start launches the hub, websocket server and begins consuming RTP packets.
func (s *Server) Start() error {
	go s.hub.Run()
	go s.serveHTTP()
	return s.consumeRTP()
}

// consumeRTP listens for incoming RTP and forwards the packets to all clients.
func (s *Server) consumeRTP() error {
	listener, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP(s.cfg.Addr), Port: s.cfg.RTPPort})
	if err != nil {
		return err
	}
	defer listener.Close()

	inbound := make([]byte, 4096)
	var once sync.Once
	fmt.Println("Waiting for RTP Packets")

	for {
		n, _, err := listener.ReadFrom(inbound)
		once.Do(func() { fmt.Print("houston we have a packet") })
		if err != nil {
			return err
		}
		pkt := &rtp.Packet{}
		if err = pkt.Unmarshal(inbound[:n]); err != nil {
			// Ignore malformed packets but continue
			continue
		}

		switch pkt.Header.PayloadType {
		case 96:
			if _, err = s.videoTrack.Write(inbound[:n]); err != nil {
				return err
			}
		case 97:
			if _, err = s.audioTrack.Write(inbound[:n]); err != nil {
				return err
			}
		}
	}
}

// serveHTTP starts the websocket endpoint used for signaling.
func (s *Server) serveHTTP() {
	http.HandleFunc("/websocket", s.websocketHandler)
	addr := s.cfg.Addr + ":" + strconv.Itoa(s.cfg.WSPort)
	if s.cfg.SSLCert != "" && s.cfg.SSLKey != "" {
		log.Fatal(http.ListenAndServeTLS(addr, s.cfg.SSLCert, s.cfg.SSLKey, nil))
	} else {
		log.Fatal(http.ListenAndServe(addr, nil))
	}
}

// websocketHandler negotiates a single WebRTC peer connection over websocket.
func (s *Server) websocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer conn.Close()

	api := s.createAPI()
	peerConnection, err := api.NewPeerConnection(s.buildConfig())
	if err != nil {
		log.Print(err)
		return
	}
	defer peerConnection.Close()

	// Attach tracks so all clients share the incoming stream.
	vtx, err := peerConnection.AddTransceiverFromTrack(s.videoTrack,
		webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionSendonly})
	atx, err := peerConnection.AddTransceiverFromTrack(s.audioTrack,
		webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionSendonly})
	if err != nil {
		log.Print(err)
		return
	}

	// Drain RTCP from senders to avoid congestion control issues.
	go func() {
		buf := make([]byte, 1500)
		for {
			if _, _, err := vtx.Sender().Read(buf); err != nil {
				return
			}
			if _, _, err := atx.Sender().Read(buf); err != nil {
				return
			}
		}
	}()

	c := ws.NewClient(s.hub, conn, peerConnection)
	go c.WriteLoop()
	s.hub.Register <- c

	peerConnection.OnICECandidate(func(i *webrtc.ICECandidate) {
		if i == nil {
			return
		}
		candidateString, err := json.Marshal(i.ToJSON())
		if err != nil {
			log.Println(err)
			return
		}
		if msg, err := json.Marshal(ws.WebsocketMessage{Event: ws.MessageTypeCandidate, Data: candidateString}); err == nil {
			s.hub.RLock()
			if _, ok := s.hub.Clients[c]; ok {
				c.Send <- msg
			}
			s.hub.RUnlock()
		}
	})

	peerConnection.OnConnectionStateChange(func(p webrtc.PeerConnectionState) {
		switch p {
		case webrtc.PeerConnectionStateFailed:
			if err := peerConnection.Close(); err != nil {
				log.Print(err)
			}
			s.hub.Unregister <- c
		case webrtc.PeerConnectionStateClosed:
			s.hub.Unregister <- c
		}
	})

	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		log.Print(err)
	}
	if err = peerConnection.SetLocalDescription(offer); err != nil {
		log.Print(err)
	}
	offerString, err := json.Marshal(offer)
	if err != nil {
		log.Print(err)
	}
	if msg, err := json.Marshal(ws.WebsocketMessage{Event: ws.MessageTypeOffer, Data: offerString}); err == nil {
		s.hub.RLock()
		if _, ok := s.hub.Clients[c]; ok {
			c.Send <- msg
		}
		s.hub.RUnlock()
	} else {
		log.Printf("could not marshal ws message: %s", err)
	}

	c.ReadLoop()
}

// createAPI configures the Pion API based on server options.
func (s *Server) createAPI() *webrtc.API {
	se := webrtc.SettingEngine{}
	if s.cfg.IP != "none" && s.cfg.ICEServers == "none" {
		se.SetNAT1To1IPs([]string{s.cfg.IP}, webrtc.ICECandidateTypeHost)
	}

	pr := strings.SplitN(s.cfg.Ports, "-", 2)
	low, _ := strconv.ParseUint(pr[0], 10, 16)
	high, _ := strconv.ParseUint(pr[1], 10, 16)
	se.SetEphemeralUDPPortRange(uint16(low), uint16(high))

	m := &webrtc.MediaEngine{}
	if err := m.RegisterDefaultCodecs(); err != nil {
		panic(err)
	}

	i := &interceptor.Registry{}
	if err := webrtc.RegisterDefaultInterceptors(m, i); err != nil {
		panic(err)
	}

	return webrtc.NewAPI(webrtc.WithMediaEngine(m), webrtc.WithInterceptorRegistry(i), webrtc.WithSettingEngine(se))
}

// buildConfig translates CLI options into a webrtc.Configuration.
func (s *Server) buildConfig() webrtc.Configuration {
	if s.cfg.ICEServers == "none" {
		return webrtc.Configuration{}
	}
	urls := strings.Split(s.cfg.ICEServers, ",")
	servers := make([]webrtc.ICEServer, len(urls))
	for idx, url := range urls {
		servers[idx] = webrtc.ICEServer{URLs: []string{url}}
	}
	return webrtc.Configuration{ICEServers: servers}
}
