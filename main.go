package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func main() {
	url, err := url.Parse("http://localhost:8080")
	if err != nil {
		panic(err)
	}
	port := flag.Int("p", 80, "port")
	flag.Parse()

	director := func(req *http.Request) {
		req.URL.Scheme = url.Scheme
		req.URL.Host = url.Host
	}

	reverseProxy := &httputil.ReverseProxy{Director: director}
	handler := handler{proxy: reverseProxy}

	http.Handle("/", handler)

	if *port == 443 {
		http.ListenAndServeTLS(fmt.Sprintf(":%d", *port), "localhost.pem", "localhost-key.pem", handler)
	} else {
		http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
	}
}

type handler struct {
	proxy *httputil.ReverseProxy
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
	h.proxy.ServeHTTP(w, r)
}
