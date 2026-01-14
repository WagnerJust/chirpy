package main

import (
	"log"
	"net/http"
)

type route struct {
	method string
	host string
	path string
	handler http.Handler
}

var routes = []route{
	{
		method: "GET",
		path: "/",
		host: "",
		handler: http.FileServer(http.Dir(".")),
	},
	{
		method: "GET",
		path: "/logo.png",
		host: "",
		handler: http.FileServer(http.Dir("./assets")),
	},
}

func addHandlers(mux *http.ServeMux) {
	for _, route := range routes {
		pattern := ""
		if route.method != "" {
			pattern += route.method + " "
		}
		if route.host != "" {
			pattern += route.host
		}
		if route.path != "" {
			pattern += route.path
		}
		mux.Handle(pattern, route.handler)
	}
}
func main () {
	serveMux := http.NewServeMux()
	addHandlers(serveMux)
	server := http.Server{
		Addr: ":8081",
		Handler: serveMux,
	}
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err.Error())
	}
	defer server.Close()


}
