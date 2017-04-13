package multiproxy

import (
  "strings"
  "net/url"
  "net/http"
  "net/http/httputil"
  "github.com/koding/websocketproxy"
)

type MultiProxy struct {
  HTTPProxy *httputil.ReverseProxy
  WebsocketProxy *websocketproxy.WebsocketProxy
}

func NewMultiProtocolSingleHostReverseProxy(host string) *MultiProxy {
  httpUrl := url.URL{
    Scheme: "http",
    Host: host,
  }
  wsUrl := url.URL{
    Scheme: "ws",
    Host: host,
  }

  rp := &MultiProxy{
    HTTPProxy: httputil.NewSingleHostReverseProxy(&httpUrl),
    WebsocketProxy: websocketproxy.NewProxy(&wsUrl),
  }
  return rp
}

func (mp *MultiProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
  if isWebsocket(req) {
    mp.WebsocketProxy.ServeHTTP(rw, req)
  } else {
    mp.HTTPProxy.ServeHTTP(rw, req)
  }
}

// https://groups.google.com/forum/#!msg/golang-nuts/KBx9pDlvFOc/QC5v-uC5UOgJ
func isWebsocket(req *http.Request) bool {
  conn_hdr := ""
  conn_hdrs := req.Header["Connection"]
  if len(conn_hdrs) > 0 {
    conn_hdr = conn_hdrs[0]
  }

  upgrade_websocket := false
  if strings.ToLower(conn_hdr) == "upgrade" {
    upgrade_hdrs := req.Header["Upgrade"]
    if len(upgrade_hdrs) > 0 {
      upgrade_websocket = (strings.ToLower(upgrade_hdrs[0]) == "websocket")
    }
  }

  return upgrade_websocket
}
