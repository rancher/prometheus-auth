package agent

import (
	"bufio"
	"fmt"
	"github.com/cockroachdb/cmux"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
)

const (
	authorizationHeaderKey = "Authorization"
)

func createHTTPListener(mux cmux.CMux) (net.Listener, error) {
	return mux.Match(
		cmux.HTTP1Fast(),
		// issue: allow anonymous to access /-/healthy, /-/ready, /metrics, /federate, /debug
		orMatcher(
			httpPathPrefixMatch("/-/healthy", "/-/ready", "/metrics", "/federate", "/debug"),
			http1HeaderFieldHas(authorizationHeaderKey, func(value string) bool {
				return strings.HasPrefix(value, "Bearer ")
			}),
		),
	), nil
}

func createGRPCListener(mux cmux.CMux, hostAccessToken string) (net.Listener, error) {
	return mux.Match(
		cmux.HTTP2(),
		httpPathPrefixMatch(v2Api),
		http2HeaderFieldEqual("content-type", "application/grpc"),
		http2HeaderFieldEqual(authorizationHeaderKey, fmt.Sprintf("Bearer %s", hostAccessToken)),
	), nil
}

func orMatcher(matchers ...cmux.Matcher) cmux.Matcher {
	return func(r io.Reader) (matched bool) {
		for _, matcher := range matchers {
			matched = matcher(r)
			if matched {
				return
			}
		}

		return
	}
}

func http1HeaderFieldHas(name string, subMatch func(value string) bool) cmux.Matcher {
	return func(r io.Reader) (matched bool) {
		req, err := http.ReadRequest(bufio.NewReader(r))
		if err != nil {
			return
		}

		value := req.Header.Get(name)
		if len(value) == 0 {
			return
		} else {
			matched = true
		}

		if subMatch != nil {
			matched = subMatch(value)
		}

		return
	}
}

func httpPathPrefixMatch(paths ...string) cmux.Matcher {
	return func(r io.Reader) (matched bool) {
		req, err := http.ReadRequest(bufio.NewReader(r))
		if err != nil {
			return
		}

		path := req.URL.Path
		for _, match := range paths {
			matched = strings.HasPrefix(path, match)
			if matched {
				return
			}
		}

		return
	}
}

func http2HeaderFieldEqual(name, value string) cmux.Matcher {
	return func(r io.Reader) (matched bool) {
		framer := http2.NewFramer(ioutil.Discard, r)
		hdec := hpack.NewDecoder(uint32(4<<10), func(hf hpack.HeaderField) {
			if strings.EqualFold(hf.Name, name) && hf.Value == value {
				matched = true
			}
		})
		for {
			f, err := framer.ReadFrame()
			if err != nil {
				return
			}

			switch f := f.(type) {
			case *http2.HeadersFrame:
				if _, err := hdec.Write(f.HeaderBlockFragment()); err != nil {
					return
				}
				if matched {
					return
				}

				if f.FrameHeader.Flags&http2.FlagHeadersEndHeaders != 0 {
					return
				}
			}
		}
	}
}
