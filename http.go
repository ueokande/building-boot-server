package main

import (
	"context"
	"log"
	"net/http"
)

type StatusCaptureResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *StatusCaptureResponseWriter) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)
	w.status = code
}

type AccessLogHandler struct {
	http.Handler
}

func (s *AccessLogHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	capw := &StatusCaptureResponseWriter{
		ResponseWriter: w,
		status:         http.StatusOK,
	}
	s.Handler.ServeHTTP(capw, req)
	log.Printf("[INFO] %s %s - %d - %s", req.Method, req.URL.Path, capw.status, req.RemoteAddr)
}

type HTTPServer struct {
	HTTPDir string

	srv *http.Server
}

func (s *HTTPServer) Start(listen string) error {
	s.srv = &http.Server{
		Addr:    listen,
		Handler: &AccessLogHandler{http.FileServer(http.Dir(s.HTTPDir))},
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
