package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"net/http/httputil"
)

func (service *Service) router(sandbox *Sandbox) http.Handler {
	r := mux.NewRouter()

	var director func(req *http.Request)

	director = func(req *http.Request) {
		req.URL.Scheme = sandbox.remoteUrl.Scheme
		req.URL.Host = sandbox.remoteUrl.Host
		req.Host = sandbox.remoteUrl.Host
	}

	proxy := &httputil.ReverseProxy{Director: director, ErrorLog: Error, ModifyResponse: func(res *http.Response) error {
		Info.Println(res)
		return nil
	}}

	r.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		newPath := fmt.Sprintf("/%s/%v%s", sandbox.name, service.cluster, r.URL.Path)

		r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", sandbox.bearerToken))
		r.Header.Set("Host", sandbox.remoteUrl.Host)
		r.Header.Set("X-Forwarded-Host", fmt.Sprintf("localhost:%d", service.port))
		r.URL.Path = newPath

		Info.Println(r)

		proxy.ServeHTTP(w, r)
		return
	})

	return r
}
