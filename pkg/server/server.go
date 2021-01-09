package server

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"path"
	"sync"

	"github.com/awnumar/memguard"
	"github.com/gorilla/mux"
)

const Version = "v0"

type Server struct {
	sync.Mutex
	r               *mux.Router
	enclavesByID    map[string]*memguard.Enclave
	maxSecretLength int64
}

func NewServer() *Server {
	memguard.CatchInterrupt()
	s := &Server{
		r:            mux.NewRouter(),
		enclavesByID: make(map[string]*memguard.Enclave),
	}
	sr := s.r.PathPrefix(path.Join("/", Version)).Subrouter()
	sr.HandleFunc("/", s.WriteSecret).Methods(http.MethodPost)
	sr.HandleFunc("/{id}", s.ReadAndDestroySecret).Methods(http.MethodGet)
	return s
}

func NewServerWithMaxSecretLength(length int64) *Server {
	s := NewServer()
	s.maxSecretLength = length
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.r.ServeHTTP(w, r)
}

func (s *Server) WriteSecret(w http.ResponseWriter, r *http.Request) {
	idBytes := make([]byte, 24)
	if _, err := rand.Read(idBytes); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	id := hex.EncodeToString(idBytes)
	var secretReader io.Reader = r.Body
	if s.maxSecretLength > 0 {
		secretReader = io.LimitReader(r.Body, s.maxSecretLength)
	}
	b, err := memguard.NewBufferFromEntireReader(secretReader)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	s.enclavesByID[id] = b.Seal()
	w.Header().Set("Location", id)
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintln(w, id)
}

func (s *Server) ReadAndDestroySecret(w http.ResponseWriter, r *http.Request) {
	id, ok := mux.Vars(r)["id"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	s.Lock()
	defer s.Unlock()
	enclave, ok := s.enclavesByID[id]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	defer delete(s.enclavesByID, id)
	b, err := enclave.Open()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer b.Destroy()
	io.Copy(w, b.Reader())
}

func (s *Server) Close() {
	memguard.Purge()
}
