package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/example/wpp-wave-bot/internal/whatsapp"
)

// Server exposes simple admin endpoints.
type Server struct {
	wa *whatsapp.Service
}

// New creates a new Server bound to the WhatsApp service.
func New(wa *whatsapp.Service) *Server {
	return &Server{wa: wa}
}

// Start runs the HTTP server on the given address.
func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/sessions", s.handleList)
	mux.HandleFunc("/sessions/", s.handleSession)
	mux.HandleFunc("/messages", s.handleSend)
	return http.ListenAndServe(addr, mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	sess := s.wa.Sessions()
	json.NewEncoder(w).Encode(map[string][]string{"sessions": sess})
}

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/sessions/"), "/")
	if len(parts) < 2 {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	id := parts[0]
	action := parts[1]
	switch action {
	case "logout":
		if r.Method != http.MethodPost && r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if err := s.wa.Logout(context.Background(), id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case "connect":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		qr, err := s.wa.Connect(r.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if qr == "" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		b64 := base64.StdEncoding.EncodeToString([]byte(qr))
		json.NewEncoder(w).Encode(map[string]string{"qr": b64})
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (s *Server) handleSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var msg whatsapp.OutgoingMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := s.wa.Send(r.Context(), &msg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}
