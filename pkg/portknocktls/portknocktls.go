package portknocktls

import (
	"crypto/tls"
	"fmt"
	"log"
	"manageportknock"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"time"
)

const pageHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta http-equiv="refresh" content="%d">
  <meta http-equiv="Cache-Control" content="no-cache, no-store, must-revalidate">
  <meta http-equiv="Pragma" content="no-cache">
  <meta http-equiv="Expires" content="0">
  <title>Portknock: %s</title>
</head>
<body>
  <h1>Portknock %s from %s</h1>
  <h2>Time: %s</h2>
  <p>Keep this page open. It refreshes every %d seconds. You may refresh manually.</p>
</body>
</html>
`

var replacerName = strings.NewReplacer("/", "", "\\", "")

var knock *manageportknock.ManagePortKnock

func decodeAndWriteResponse(w http.ResponseWriter, r *http.Request) {
	if knock == nil {
		http.NotFound(w, r)
		return
	}

	portKnockName := replacerName.Replace(strings.Trim(r.URL.Path, "/"))
	if !knock.CheckName(portKnockName) {
		http.NotFound(w, r)
		return
	}

	remoteHost, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		if addr, parseErr := netip.ParseAddr(remoteHost); parseErr == nil {
			knock.Insert(portKnockName, addr)
		}
	}
	refresh := int(float64(knock.Timeout()) / 2.2)
	if refresh < 1 {
		refresh = 1
	}
	timeNow := time.Now().Format(time.DateTime)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("Refresh", strconv.Itoa(refresh))
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, pageHTML, refresh, portKnockName,
		portKnockName, r.RemoteAddr, timeNow, refresh)
}

func GoStartTLSPortKnock(config *manageportknock.ManagePortKnock) {

	if config == nil || !config.IsConfigAndContainsKnocks() {
		return
	}
	knock = config
	addr := fmt.Sprintf(":%d", config.TLSPort())
	certFile := config.TLSCertFile()
	keyFile := config.TLSKeyFile()

	mux := http.NewServeMux()
	for name := range config.Iterator() {
		mux.HandleFunc("/"+name, decodeAndWriteResponse)
	}

	srv := &http.Server{
		Addr:           addr,
		Handler:        mux,
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   5 * time.Second,
		MaxHeaderBytes: 4096,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	log.Printf("HTTPS server listening on %s", addr)
	if err := srv.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
		log.Fatalf("PortKnock: listen and serve TLS: %v", err)
	}

}
