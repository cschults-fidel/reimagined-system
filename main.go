// Command reimagined-system serves a minimal, in-memory deployments API.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"
)

// Deployment is a single deployment record.
type Deployment struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Image     string    `json:"image"`
	Replicas  int       `json:"replicas"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// createRequest is the body accepted by POST /deployments.
type createRequest struct {
	Name     string `json:"name"`
	Image    string `json:"image"`
	Replicas *int   `json:"replicas"` // pointer so we can tell "omitted" from 0
}

// Store keeps deployments in memory. It is safe for concurrent use.
type Store struct {
	mu          sync.RWMutex
	nextID      int
	deployments map[string]Deployment
}

func NewStore() *Store {
	return &Store{deployments: make(map[string]Deployment)}
}

func (s *Store) Create(name, image string, replicas int) Deployment {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	d := Deployment{
		ID:        strconv.Itoa(s.nextID),
		Name:      name,
		Image:     image,
		Replicas:  replicas,
		Status:    "pending",
		CreatedAt: time.Now().UTC(),
	}
	s.deployments[d.ID] = d
	return d
}

func (s *Store) Get(id string) (Deployment, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	d, ok := s.deployments[id]
	return d, ok
}

// List returns all deployments, oldest first.
func (s *Store) List() []Deployment {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Deployment, 0, len(s.deployments))
	for _, d := range s.deployments {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool {
		a, _ := strconv.Atoi(out[i].ID)
		b, _ := strconv.Atoi(out[j].ID)
		return a < b
	})
	return out
}

// Server wires HTTP handlers to a Store.
type Server struct {
	store *Store
}

func NewServer(store *Store) http.Handler {
	s := &Server{store: store}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /deployments", s.createDeployment)
	mux.HandleFunc("GET /deployments", s.listDeployments)
	mux.HandleFunc("GET /deployments/{id}", s.getDeployment)
	return mux
}

func (s *Server) createDeployment(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB

	var req createRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	replicas, err := validate(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	d := s.store.Create(req.Name, req.Image, replicas)
	w.Header().Set("Location", "/deployments/"+d.ID)
	writeJSON(w, http.StatusCreated, d)
}

func (s *Server) listDeployments(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.List())
}

func (s *Server) getDeployment(w http.ResponseWriter, r *http.Request) {
	d, ok := s.store.Get(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, "deployment not found")
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// validate checks a create request and returns the replica count to use.
func validate(req createRequest) (int, error) {
	if req.Name == "" {
		return 0, errors.New("name is required")
	}
	if req.Image == "" {
		return 0, errors.New("image is required")
	}
	if req.Replicas == nil {
		return 1, nil
	}
	if *req.Replicas < 0 {
		return 0, fmt.Errorf("replicas must be >= 0, got %d", *req.Replicas)
	}
	return *req.Replicas, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func main() {
	addr := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		addr = ":" + port
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           NewServer(NewStore()),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
