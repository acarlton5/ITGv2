// Package server implements the WebRTC signaling and RTP handling logic.
package server

import (
	"encoding/json"
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
type Stream struct {
	port       int
	videoTrack *webrtc.TrackLocalStaticRTP
	audioTrack *webrtc.TrackLocalStaticRTP
	hub        *ws.Hub
	conn       *net.UDPConn
}

type Server struct {
	cfg      Config
	upgrader websocket.Upgrader
	streams  map[int]*Stream
	mu       sync.Mutex
}

// New creates a new server instance using the provided configuration.
func New(cfg Config) (*Server, error) {
	s := &Server{
		cfg:      cfg,
		upgrader: websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		streams:  make(map[int]*Stream),
	}

	if _, err := s.createStream(cfg.RTPPort); err != nil {
		return nil, err
	}
	return s, nil
}

// Start launches the hub, websocket server and begins consuming RTP packets.
func (s *Server) Start() error {
	go s.serveHTTP()
	return nil
}

// serveHTTP starts the websocket endpoint used for signaling.
func (s *Server) serveHTTP() {
	http.HandleFunc("/websocket", s.websocketHandler)
	http.HandleFunc("/stream", s.streamHandler)
	addr := s.cfg.Addr + ":" + strconv.Itoa(s.cfg.WSPort)
	if s.cfg.SSLCert != "" && s.cfg.SSLKey != "" {
		log.Fatal(http.ListenAndServeTLS(addr, s.cfg.SSLCert, s.cfg.SSLKey, nil))
	} else {
		log.Fatal(http.ListenAndServe(addr, nil))
	}
}

// streamHandler creates a new stream on demand. The optional `port` query
// parameter allows specifying the UDP port to listen on. If omitted or zero, a
// random port is chosen. It responds with a JSON object containing the actual
// port in use.
func (s *Server) streamHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		port := 0
		if p := r.URL.Query().Get("port"); p != "" {
			if v, err := strconv.Atoi(p); err == nil {
				port = v
			}
		}

		st, err := s.createStream(port)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]int{"port": st.port})
	case http.MethodDelete:
		p := r.URL.Query().Get("port")
		if p == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		port, err := strconv.Atoi(p)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		s.deleteStream(port)
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// websocketHandler negotiates a single WebRTC peer connection over websocket.
func (s *Server) websocketHandler(w http.ResponseWriter, r *http.Request) {
	port := s.cfg.RTPPort
	if p := r.URL.Query().Get("port"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			port = v
		}
	}

	s.mu.Lock()
	stream, ok := s.streams[port]
	s.mu.Unlock()
	if !ok {
		http.Error(w, "unknown stream", http.StatusNotFound)
		return
	}

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

	// Attach tracks for this stream.
	vtx, err := peerConnection.AddTransceiverFromTrack(stream.videoTrack,
		webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionSendonly})
	atx, err := peerConnection.AddTransceiverFromTrack(stream.audioTrack,
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

	c := ws.NewClient(stream.hub, conn, peerConnection)
	go c.WriteLoop()
	stream.hub.Register <- c

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
			stream.hub.RLock()
			if _, ok := stream.hub.Clients[c]; ok {
				c.Send <- msg
			}
			stream.hub.RUnlock()
		}
	})

	peerConnection.OnConnectionStateChange(func(p webrtc.PeerConnectionState) {
		switch p {
		case webrtc.PeerConnectionStateFailed:
			if err := peerConnection.Close(); err != nil {
				log.Print(err)
			}
			stream.hub.Unregister <- c
		case webrtc.PeerConnectionStateClosed:
			stream.hub.Unregister <- c
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
		stream.hub.RLock()
		if _, ok := stream.hub.Clients[c]; ok {
			c.Send <- msg
		}
		stream.hub.RUnlock()
	} else {
		log.Printf("could not marshal ws message: %s", err)
	}

	c.ReadLoop()
}

// createStream initializes a new Stream listening on the provided port. If port
// is 0, the system will choose an available one. The new Stream begins
// consuming RTP immediately.
func (s *Server) createStream(port int) (*Stream, error) {
	l, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP(s.cfg.Addr), Port: port})
	if err != nil {
		return nil, err
	}

	v, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: "video/H264"}, "video", "pion")
	if err != nil {
		l.Close()
		return nil, err
	}
	a, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: "audio/opus"}, "audio", "pion")
	if err != nil {
		l.Close()
		return nil, err
	}

	st := &Stream{port: l.LocalAddr().(*net.UDPAddr).Port, videoTrack: v, audioTrack: a, hub: ws.NewHub(), conn: l}
	s.mu.Lock()
	s.streams[st.port] = st
	s.mu.Unlock()

	go st.hub.Run()
	go func(stream *Stream) {
		defer stream.conn.Close()
		inbound := make([]byte, 4096)
		for {
			n, _, err := stream.conn.ReadFrom(inbound)
			if err != nil {
				return
			}
			pkt := &rtp.Packet{}
			if err = pkt.Unmarshal(inbound[:n]); err != nil {
				continue
			}
			switch pkt.Header.PayloadType {
			case 96:
				_, _ = stream.videoTrack.Write(inbound[:n])
			case 97:
				_, _ = stream.audioTrack.Write(inbound[:n])
			}
		}
	}(st)

	return st, nil
}

// deleteStream closes and removes the stream associated with the UDP port.
func (s *Server) deleteStream(port int) {
	s.mu.Lock()
	st, ok := s.streams[port]
	if ok {
		delete(s.streams, port)
	}
	s.mu.Unlock()
	if ok {
		st.conn.Close()
		st.hub.Close()
	}
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
