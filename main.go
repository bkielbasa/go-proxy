package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"
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
	for name, _ := range r.Header {
		if strings.ToLower(name) == "set-cookie" {
			// For example keep one domain unchanged, rewrite one domain and remove other domains
			cookieConfig := make(map[string]string)
			cookieConfig["unchanged.domain"] = "unchanged.domain"
			cookieConfig["old.domain"] = "new.domain"
			cookieConfig["google.com"] = "localhost"
			//remove other cookies
			cookieConfig["*"] = ""
			r.Header.Set(name, rewriteCookieDomain(r.Header.Get(name), cookieConfig))
		}
	}
	r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
	h.proxy.ServeHTTP(w, r)
}


// https://gist.github.com/elliotchance/d419395aa776d632d897
func ReplaceAllStringSubmatchFunc(re *regexp.Regexp, str string, repl func([]string) string) string {
	result := ""
	lastIndex := 0

	for _, v := range re.FindAllSubmatchIndex([]byte(str), -1) {
		groups := []string{}
		for i := 0; i < len(v); i += 2 {
			groups = append(groups, str[v[i]:v[i+1]])
		}

		result += str[lastIndex:v[0]] + repl(groups)
		lastIndex = v[1]
	}

	return result + str[lastIndex:]
}

/* config is mapping of domains to new domains, use "*" to match all domains.
For example keep one domain unchanged, rewrite one domain and remove other domains:
cookieDomainRewrite: {
  "unchanged.domain": "unchanged.domain",
  "old.domain": "new.domain",
  "*": ""
*/
func rewriteCookieDomain(header string, config map[string]string) string {

	re := regexp.MustCompile(`(?i)(\s*; Domain=)([^;]+)`)
	return ReplaceAllStringSubmatchFunc(re, header, func(groups []string) string {
		match, prefix, previousValue := groups[0], groups[1], groups[2]

		var newValue string
		if config[previousValue] != "" {
			newValue = config[previousValue]
		} else if config["*"] != "" {
			newValue = config["*"]
		} else {
			//no match, return previous value
			return match
		}
		if newValue != "" {
			//replace value
			return prefix + newValue
		} else {
			//remove value
			return ""
		}
	})
}
