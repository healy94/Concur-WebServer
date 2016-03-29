package main

import (
	"net"
	"time"
	"sync"
)

type Handler interface {
	ServeHTTP(ResponseWriter, *Request)
}

type Header map[string][]string

// criterion for HTTP server 
type Server struct {
	Addr 			string
	Handler 		Handler
	ReadTimeout 	time.Duration
	WriteTimeout 	time.Duration
	MaxHeaderBytes 	int
	nextProtoErr	error
	nextProtoOnce	sync.Once

}

// sets the TCP timeouts on the connections that are accepted
type tcpKeepAliveListener struct {
	*net.TCPListner
}

type ResponseWriter interface {
	Header() Header
	Write([]byte) (int,error) // writes data to connection as the HTTP reply
	WriteHeader(int)	// sends HTTP response
}

// StateNew represents a new connection that is expected to send a request
var StateNew int = iota
var testHookServerServe func(*Server, net.Listener)

/* Function and Method "ListenAndServe" gets an address from the Transmission Control Protocol(TCP)
	and calls function "Serve" with a given handler to handle the client requests */
func ListenAndServe(address string, handler Handler) error {
	server := &Server{Addr: address, Handler: handler}
	return server.ListenAndServe()
}

func (server * Server) ListenAndServe() error {
	address := server.Addr

	//if the server address is blank then ":http" is used for the requests
	if address == "" {
		address = ":http"
	}

	listener,err := net.Listen("tcp",address)
	// errors should always equal nil for the function to run properly
	// the "err" does not equal nil the function fails, and the error is returned
	if err != nil{
		return err
	}

	return server.Serve(tcpKeepAliveListener{listener.(*net.TCPListner)})
}

/* Function  and Method "Serve" takes incoming HTTP connections on listener and makes goroutines for each
	listener. These goroutines then call handler to serve each */
func Serve(listener net.Listener, handler Handler) error {
	server := &Server{Handler: handler}
	return server.Serve(listener)
}

func (server *Server) Serve(listener net.Listener) error {
	defer listener.Close()
	if fn := testHookServerServe; fn != nil {
		fn(server, listener)
	}

	// temperary delay for time duration of sleep on failure
	var tDel time.Duration

	if err := server.setupHTTP2(); err != nil {
		return err
	}

	for {
		rw, err := listener.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tDel == 0 {
					tDel = 5 * time.Millisecond
				} else {
					tDel *= 2
				}
				if max := 1 * time.Second; tDel > max {
					tDel = max
				}

				server.logf("http: Accept failure: %v; try again in %v",err,tDel)
				time.Sleep(tDel)
				continue
			}
			return err
		}
		tDel = 0
		c := server.newConn(rw)
		c.setState(c.rwc, StateNew)
		go c.serve()
	} 
}

// log errors
func (s *Server) logf(format string, args ...interface{}) {
		if s.ErrorLog != nil {
			s.ErrorLog.Printf(format, args...)
		} else {
			log.Printf(format, args...)
		}
}

// create a new connection
func (srv *Server) newConn(rwc net.Conn) *conn {
		c := &conn{
			server: srv,
  			rwc:    rwc,
   		}
   		if debugServerConnections {
   			c.rwc = newLoggingConn("server", c.rwc)
   		}
   		return c
}

func (server *Server) setupHTTP2() error {
  server.nextProtoOnce.Do(server.onceSetNextProtoDefaults)
  return server.nextProtoErr
}



