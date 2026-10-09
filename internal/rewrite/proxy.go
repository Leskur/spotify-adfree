package rewrite

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"golang.org/x/net/http2"

	"github.com/leskur/spotify-ad-free/internal/rules"
	"github.com/leskur/spotify-ad-free/internal/sysproxy"
)

const DefaultPort = 18513

// Charles Rewrite：拨无效域名时的等待。
const rewriteDialTimeout = 75 * time.Second

type Server struct {
	Addr  string
	Rules *rules.Rules
	CA    *CA

	server    *http.Server
	certMu    sync.Mutex
	certCache map[string]*tls.Certificate
}

func NewServer(addr string, r *rules.Rules, ca *CA) *Server {
	return &Server{
		Addr:      addr,
		Rules:     r,
		CA:        ca,
		certCache: map[string]*tls.Certificate{},
	}
}

func (s *Server) Start() error {
	s.server = &http.Server{
		Addr:              s.Addr,
		Handler:           http.HandlerFunc(s.handle),
		ReadHeaderTimeout: 10 * time.Second,
	}
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	s.Addr = ln.Addr().String()
	go func() { _ = s.server.Serve(ln) }()
	return nil
}

func (s *Server) Stop() error {
	if s.server == nil {
		return nil
	}
	return s.server.Close()
}

func (s *Server) Port() int {
	_, port, err := net.SplitHostPort(s.Addr)
	if err != nil {
		return DefaultPort
	}
	var p int
	_, _ = fmt.Sscanf(port, "%d", &p)
	return p
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodConnect {
		http.Error(w, "CONNECT only", http.StatusMethodNotAllowed)
		return
	}
	s.handleConnect(w, r)
}

func (s *Server) handleConnect(w http.ResponseWriter, r *http.Request) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijack unsupported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hj.Hijack()
	if err != nil {
		return
	}

	hostname := r.Host
	if h, _, err := net.SplitHostPort(r.Host); err == nil {
		hostname = h
	}

	if _, err := clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
		_ = clientConn.Close()
		return
	}

	tlsCert, err := s.tlsCertFor(hostname)
	if err != nil {
		_ = clientConn.Close()
		return
	}

	tlsClient := tls.Server(clientConn, &tls.Config{
		Certificates: []tls.Certificate{*tlsCert},
		NextProtos:   []string{"h2", "http/1.1"},
	})
	if err := tlsClient.Handshake(); err != nil {
		_ = clientConn.Close()
		return
	}

	// 在这条 MITM 连接上跑 HTTP(S) 服务：
	// - Block：ErrAbortHandler → HTTP/2 只重置当前 stream，不掐整条连接（避免首播挂掉）
	// - 其它：反代到真实上游
	handler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		path := req.URL.Path
		if path == "" {
			path = "/"
		}

		if s.Rules.IsBlocked(hostname, path) {
			// 不写响应体；HTTP/2 下尽量只打掉当前请求
			panic(http.ErrAbortHandler)
		}

		if redir := s.Rules.RedirectHost(hostname, path); redir != "" {
			s.serveRewrite(rw, req, redir)
			return
		}

		s.serveProxy(rw, req, hostname)
	})

	mitm := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 15 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	_ = http2.ConfigureServer(mitm, &http2.Server{})

	ln := newOnceListener(tlsClient)
	_ = mitm.Serve(ln)
}

func (s *Server) serveProxy(rw http.ResponseWriter, req *http.Request, hostname string) {
	target, _ := url.Parse("https://" + hostname)
	listenPort := s.Port()
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = &http.Transport{
		Proxy: func(r *http.Request) (*url.URL, error) {
			return sysproxy.HTTP(r, listenPort)
		},
		TLSClientConfig: &tls.Config{
			ServerName: hostname,
			NextProtos: []string{"h2", "http/1.1"},
		},
		ForceAttemptHTTP2:     true,
		ResponseHeaderTimeout: 60 * time.Second,
		IdleConnTimeout:       90 * time.Second,
	}
	originalDirector := proxy.Director
	proxy.Director = func(r *http.Request) {
		originalDirector(r)
		r.Host = hostname
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		panic(http.ErrAbortHandler)
	}
	proxy.ServeHTTP(rw, req)
}

func (s *Server) serveRewrite(rw http.ResponseWriter, req *http.Request, redirHost string) {
	// 拨无效域名并等待，不回包 —— 让客户端自己超时
	dialer := &net.Dialer{Timeout: rewriteDialTimeout}
	conn, err := tls.DialWithDialer(dialer, "tcp", net.JoinHostPort(redirHost, "443"), &tls.Config{
		ServerName: redirHost,
	})
	if err != nil {
		panic(http.ErrAbortHandler)
	}
	_ = conn.Close()
	panic(http.ErrAbortHandler)
}

func (s *Server) tlsCertFor(host string) (*tls.Certificate, error) {
	s.certMu.Lock()
	defer s.certMu.Unlock()
	if c, ok := s.certCache[host]; ok {
		return c, nil
	}
	certPEM, keyPEM, err := s.CA.Issue(host)
	if err != nil {
		return nil, err
	}
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}
	s.certCache[host] = &cert
	return &cert, nil
}

// onceListener 把单条已建立的连接伪装成 Listener，供 http.Server.Serve。
type onceListener struct {
	addr net.Addr
	ch   chan net.Conn
}

func newOnceListener(c net.Conn) *onceListener {
	ch := make(chan net.Conn, 1)
	ch <- c
	return &onceListener{addr: c.LocalAddr(), ch: ch}
}

func (l *onceListener) Accept() (net.Conn, error) {
	c, ok := <-l.ch
	if !ok {
		return nil, net.ErrClosed
	}
	return c, nil
}

func (l *onceListener) Close() error {
	select {
	case <-l.ch:
	default:
	}
	close(l.ch)
	return nil
}

func (l *onceListener) Addr() net.Addr { return l.addr }
