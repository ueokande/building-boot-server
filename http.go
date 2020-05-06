package main

import (
	"context"
	"log"
	"net/http"
)

type HTTPServer struct {
	HTTPDir string

	srv *http.Server
}

func (s *HTTPServer) Start(listen string) error {
	s.srv = &http.Server{
		Addr:    listen,
		Handler: http.FileServer(http.Dir(s.HTTPDir)),
	}

	log.Printf("[INFO] Starting HTTP server on %s ...", listen)
	err := s.srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *HTTPServer) Shutdown() error {
	return s.srv.Shutdown(context.TODO())
}
