package myhttp

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type request struct {
	useHTTPS bool
	Header   string
	addr     string
}

// Response is used for workers to store one HTTP request results
type Response struct {
	Status       string
	ResponseBody string
	StatusCode   int
	ResponseSize int64
	ResponseTime int64
}

// Client keep the connection and request website
type Client struct {
	HTTP10  bool
	Verbose bool
	Conn    net.Conn
}

type responseHandler struct {
	chunked         bool  // Transfer-Encoding: chunked
	shouldCloseConn bool  // Connection: close or keep-alive
	contentLength   int64 // Content-Length: int
	verbose         bool
	br              *bufio.Reader
}

func connect(ctx context.Context, r *request) (conn net.Conn, err error) {
	ch := make(chan bool)
	go func() {
		if r.useHTTPS {
			d := &tls.Dialer{}
			conn, err = d.DialContext(ctx, "tcp", r.addr)
		} else {
			d := &net.Dialer{}
			conn, err = d.DialContext(ctx, "tcp", r.addr)
		}
		ch <- true
	}()

	select {
	case <-ch:
		return
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

type connWithCounter struct {
	reader     io.Reader
	totalBytes int64
}

func (cc *connWithCounter) Read(p []byte) (int, error) {
	n, err := cc.reader.Read(p)
	cc.totalBytes += int64(n)
	return n, err
}

// GET request the url with HTTP GET
func (client *Client) GET(url string) (*Response, error) {
	var err error
	var rc *Response
	request, err := client.newRequest(url)
	if client.Verbose {
		fmt.Print(request.Header)
	}
	finalize := func() (*Response, error) {
		if client.Conn != nil && client.HTTP10 {
			client.Conn.Close()
			client.Conn = nil
		}
		return nil, err
	}
	if err != nil {
		return finalize()
	}
	if client.Conn == nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		client.Conn, err = connect(ctx, request)
		if err != nil {
			return finalize()
		}
	}

	tRequestTime := time.Now()
	_, err = client.Conn.Write([]byte(request.Header))
	if err != nil {
		return finalize()
	}

	rc, err = client.ReadResponse()
	if err != nil {
		return finalize()
	}
	_, err = finalize()
	rc.ResponseTime = time.Since(tRequestTime).Milliseconds()

	return rc, err
}

func (client *Client) newRequest(reqURL string) (*request, error) {
	request := &request{useHTTPS: true}
	if strings.HasPrefix(reqURL, "http://") {
		request.useHTTPS = false
	} else if !strings.HasPrefix(reqURL, "https://") {
		reqURL = "https://" + reqURL
	}

	u, err := url.Parse(reqURL)
	if err != nil {
		return nil, err
	}

	port := "443"
	if u.Port() != "" {
		port = u.Port()
	} else {
		if !request.useHTTPS {
			port = "80"
		}
	}

	request.addr = u.Hostname() + ":" + port
	httpVersion := "1.1"
	if client.HTTP10 {
		httpVersion = "1.0"
	}
	path := u.Path
	if !strings.HasSuffix(path, "/") && !strings.ContainsAny(path, ".") {
		path = path + "/"
	}
	request.Header = fmt.Sprint(
		"GET "+path+" HTTP/"+httpVersion+"\r\n",
		"HOST: "+u.Hostname()+"\r\n",
		"User-Agent: ngoperf/0.1.0 \r\n",
		"Accept: */*\r\n",
		"\r\n",
	)
	return request, nil
}

func readLine(br *bufio.Reader) ([]byte, error) {
	var line []byte
	for {
		l, more, err := br.ReadLine()
		if err != nil {
			return nil, err
		}
		if line == nil && !more {
			return l, nil
		}
		line = append(line, l...)
		if !more {
			break
		}
	}
	return line, nil
}

func (handler *responseHandler) readResponseBody(r *Response) ([]byte, error) {
	var reader io.Reader
	if handler.shouldCloseConn { // http 1.0
		reader = handler.br
	} else if handler.contentLength > 0 {
		reader = io.LimitReader(handler.br, handler.contentLength)
	} else if handler.chunked {
		reader = NewChunkedReader(handler.br)
	} else { // no content
		return []byte{}, nil
	}

	var body []byte
	for { // read until io.EOF
		var buffer = make([]byte, 4096)
		n, err := reader.Read(buffer)
		if n > 0 && (err == nil || err == io.EOF) {
			body = append(body, buffer[:n]...)
		}
		if err != nil {
			if err == io.EOF && n == 0 {
				err = nil
			}
			return body, nil
		}
	}
}

// ReadResponse read client.conn to Response
func (client *Client) ReadResponse() (r *Response, err error) {
	cc := &connWithCounter{reader: client.Conn}
	handler := &responseHandler{br: bufio.NewReader(cc), verbose: client.Verbose}
	r = &Response{}
	if err = handler.readStatusLine(r); err != nil {
		return nil, err
	}

	if err = handler.readHeader(r); err != nil {
		return nil, err
	}

	var responseBody []byte
	responseBody, err = handler.readResponseBody(r)
	if err != nil {
		return nil, err
	}

	// connection: close in Response header
	if handler.shouldCloseConn {
		client.Conn.Close()
		client.Conn = nil
	}
	r.ResponseBody = string(responseBody)
	r.ResponseSize = cc.totalBytes

	return r, nil

}

func (handler *responseHandler) readStatusLine(r *Response) error {
	line, err := readLine(handler.br)
	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return err
	}
	var i int
	stringLine := string(line)
	if handler.verbose {
		fmt.Println(stringLine)
	}
	if i = strings.IndexByte(stringLine, ' '); i == -1 {
		return errors.New("Invalid HTTP response: " + stringLine)
	}
	r.Status = strings.TrimSpace(stringLine[i+1:])
	status := ""
	if i = strings.IndexByte(r.Status, ' '); i != -1 {
		status = r.Status[:i]
	}
	if len(status) != 3 {
		return errors.New("Invalid HTTP status: " + status)
	}
	r.StatusCode, err = strconv.Atoi(status)
	if err != nil || r.StatusCode < 0 {
		return errors.New("Invalid HTTP status code: " + string(r.StatusCode))
	}
	return nil
}

func (handler *responseHandler) readHeader(r *Response) error {
	for {
		kv, err := readLine(handler.br)
		if handler.verbose {
			fmt.Println(string(kv))
		}
		if len(kv) == 0 {
			break
		}
		i := bytes.IndexByte(kv, ':')
		if i < 0 {
			continue
		}

		key := string(bytes.TrimSpace(kv[:i]))
		i++ // skip colon
		for i < len(kv) && (kv[i] == ' ' || kv[i] == '\t') {
			i++
		}
		val := string(bytes.TrimSpace(kv[i:]))
		key = strings.ToLower(key)
		val = strings.ToLower(val)
		switch key {
		case "content-length":
			handler.contentLength, err = strconv.ParseInt(val, 10, 64)
			if err != nil {
				return err
			}
		case "transfer-encoding":
			if val != "chunked" {
				return errors.New("Only support Transfer-Encoding: chunked")
			}
			handler.contentLength = -1
			handler.chunked = true
		case "connection":
			if val == "close" {
				handler.shouldCloseConn = true
			}
		case "trailer":
			// RFC 7230, section 4.1.2: Chunked trailer part
			return errors.New("Chunked trailer part not implement")
		}
	}
	return nil
}
