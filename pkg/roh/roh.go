// package roh provides a simple RPC over HTTP(s)
// It works using convention over configuration.
// For normal RPC calls, only POST is used (since QUERY is too new yet).
// The POST posts a JSON body with the query to a method path,
// And the response is equally a JSON body, even on error.
// GET queries are allowed to serve files.
package roh

import "bytes"
import "net/http"
import "net/url"
import "net"
import "encoding/json"
import "log/slog"

type Server struct {
	*http.Server
	*http.ServeMux
	State chan (http.ConnState)
}

func NewServer(hostport string) *Server {
	s := &Server{}
	s.Server = &http.Server{}
	s.Addr = hostport
	s.ServeMux = http.NewServeMux()
	s.Server.Handler = s.ServeMux
	s.State = make(chan (http.ConnState))
	s.Server.ConnState = func(conn net.Conn, c http.ConnState) {
		select {
		case s.State <- c:
		default:
		}
	}
	return s
}

type JSONHandler struct {
	cb func(body []byte) (result []byte, err error)
}

func (j JSONHandler) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	ct := req.Header.Get("Content-Type")
	if ct != "application/json" {
		wr.WriteHeader(http.StatusBadRequest)
		return
	}

	var buf []byte
	_, err := req.Body.Read(buf)
	if err != nil {
		wr.WriteHeader(http.StatusBadRequest)
		return
	}

	result, err := j.cb(buf)
	if err != nil {
		wr.WriteHeader(http.StatusBadRequest)
	}
	wr.Header().Set("Content-Type", "application/json")
	_, err = wr.Write(result)
	if err != nil {
		wr.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *Server) RegisterJSON(path string, cb func(body []byte) (result []byte, err error)) {
	handler := &JSONHandler{cb: cb}
	s.ServeMux.Handle("POST "+path, handler)
}

type TypeHandler[T, U any] struct {
	cb func(body T) (result U, err error)
}

func (t TypeHandler[T, U]) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	slog.Info("Request", "req", req)
	ct := req.Header.Get("Content-Type")
	if ct != "application/json" {
		wr.WriteHeader(http.StatusBadRequest)
		return
	}

	var buf []byte
	_, err := req.Body.Read(buf)
	if err != nil {
		wr.WriteHeader(http.StatusBadRequest)
		wr.Write([]byte(err.Error()))
		return
	}
	var request T
	err = json.Unmarshal(buf, &request)
	if err != nil {
		wr.WriteHeader(http.StatusBadRequest)
		wr.Write([]byte(err.Error()))
		return
	}

	response, err := t.cb(request)
	if err != nil {
		wr.WriteHeader(http.StatusBadRequest)
		wr.Write([]byte(err.Error()))
		return
	}

	result, err := json.Marshal(response)
	if err != nil {
		wr.WriteHeader(http.StatusInternalServerError)
		wr.Write([]byte(err.Error()))
		return
	}

	wr.Header().Set("Content-Type", "application/json")

	_, err = wr.Write(result)
	if err != nil {
		wr.WriteHeader(http.StatusInternalServerError)
	}
}

func Register[T, U any](s *Server, path string, cb func(T) (U, error)) {
	handler := &TypeHandler[T, U]{cb: cb}
	s.ServeMux.Handle("POST "+path, handler)
}

type Client struct {
	http.Client
	target *url.URL
}

func NewClient(target string) (*Client, error) {
	var err error
	c := &Client{}
	c.target, err = url.Parse(target)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func Call[U, T any](c *Client, path string, request T) (U, error) {
	var result U
	target := c.target.JoinPath(path)

	rbuf, err := json.Marshal(request)
	if err != nil {
		return result, err
	}
	sbuf := bytes.NewBuffer(rbuf)

	res, err := c.Post(target.String(), "application/json", sbuf)
	var buf []byte
	_, err = res.Body.Read(buf)
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(buf, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}
