package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"net/http/httputil"
)

const sandboxEnvironmentProdDomain = "sandbox.pdok.nl"
const sandboxEnvironmentDevDomain = "localhost:32788"

func (service *service) router(sandbox *sandbox) http.Handler {
	r := mux.NewRouter()

	var director func(req *http.Request)

	if sandbox.dev {
		director = func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = sandboxEnvironmentDevDomain
		}
	} else {
		director = func(req *http.Request) {
			req.URL.Scheme = "https"
			req.URL.Host = sandboxEnvironmentProdDomain
		}
	}

	proxy := &httputil.ReverseProxy{Director: director, ErrorLog: Error, ModifyResponse: func(res *http.Response) error {
		Info.Println(res)
		return nil
	}}

	r.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		newPath := fmt.Sprintf("/%s/%v%s", sandbox.name, service.cluster, r.URL.Path)

		r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", sandbox.bearerToken))
		r.URL.Path = newPath

		Info.Println(r)

		proxy.ServeHTTP(w, r)
		return
	})

	return r
}
