package main

import (
	"github.com/gorilla/mux"
	"net/http"
)

func (service *service) router() http.Handler {
	r := mux.NewRouter()
	return r
}
