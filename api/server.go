package api

import (
	"fmt"
	"github.com/mitchellh/cli"
	"github.com/ryotarai/spotscaler/state"
	"net/http"
)

type Server struct {
	Ui    cli.Ui
	State state.State
	Addr  string
}

func (server *Server) Start() error {
	s := &http.Server{
		Addr:    server.Addr,
		Handler: server,
	}
	server.Ui.Info(fmt.Sprintf("Starting HTTP API server on %q", server.Addr))
	go func() {
		err := s.ListenAndServe()
		if err != nil {
			server.Ui.Error(fmt.Sprintf("ListenAndServe error: %s", err))
		}
	}()

	return nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Ui.Output(fmt.Sprint(*r))
}
