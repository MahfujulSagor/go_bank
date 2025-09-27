package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// APIServer serves HTTP requests for the banking service.
// It uses Gorilla Mux for routing.
type APIServer struct {
	Addr  string
	store Storage
}

// NewAPIServer creates a new APIServer with the given address.
// The address is in the form ":port".
func NewAPIServer(addr string, store Storage) *APIServer {
	return &APIServer{
		Addr:  addr,
		store: store,
	}
}

// Start starts the HTTP server and listens for requests.
// It sets up the routes and handlers.
func (s *APIServer) Start() {
	router := mux.NewRouter()

	router.HandleFunc("/account", makeHTTPHandleFunc(s.handleAccount))
	router.HandleFunc("/account/{id}", makeHTTPHandleFunc(s.handleGetAccountByID))

	log.Println("Starting server on", s.Addr)
	http.ListenAndServe(s.Addr, router)
}

func (s *APIServer) handleAccount(w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case http.MethodGet:
		return s.handleGetAccount(w, r)
	case http.MethodPost:
		return s.handleCreateAccount(w, r)
	case http.MethodDelete:
		return s.handleDeleteAccount(w, r)
	default:
		return fmt.Errorf("method not allwed %s", r.Method)
	}
}

// handleGetAccount handles the GET /account request.
// It retrieves all accounts from the storage and returns them as JSON.
// Limited to 10 accounts for simplicity.
func (s *APIServer) handleGetAccount(w http.ResponseWriter, r *http.Request) error {
	accounts, err := s.store.GetAccounts()
	if err != nil {
		return writeJSON(w, http.StatusInternalServerError, ApiError{Error: "failed to get accounts"})
	}
	if len(accounts) < 1 {
		return writeJSON(w, http.StatusNotFound, ApiError{Error: "no accounts found"})
	}

	return writeJSON(w, http.StatusOK, accounts)
}

func (s *APIServer) handleGetAccountByID(w http.ResponseWriter, r *http.Request) error {
	idStr := mux.Vars(r)["id"]
	if idStr == "" {
		return writeJSON(w, http.StatusBadRequest, ApiError{Error: "missing account id"})
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return writeJSON(w, http.StatusBadRequest, ApiError{Error: "invalid account id"})
	}

	account, err := s.store.GetAccountByID(id)
	if err != nil {
		return writeJSON(w, http.StatusInternalServerError, ApiError{Error: "failed to get account"})
	}
	if account == nil {
		return writeJSON(w, http.StatusNotFound, ApiError{Error: "account not found"})
	}

	return writeJSON(w, http.StatusOK, account)
}

// handleCreateAccount handles the POST /account request.
func (s *APIServer) handleCreateAccount(w http.ResponseWriter, r *http.Request) error {
	createAccountReq := &CreateAccountRequest{}
	if err := json.NewDecoder(r.Body).Decode(createAccountReq); err != nil {
		return writeJSON(w, http.StatusBadRequest, ApiError{Error: "invalid request payload"})
	}

	account := NewAccount(createAccountReq.FirstName, createAccountReq.LastName)
	id, err := s.store.CreateAccount(account)
	if err != nil {
		return writeJSON(w, http.StatusInternalServerError, ApiError{Error: "failed to create accont"})
	}
	return writeJSON(w, http.StatusCreated, map[string]int64{
		"id": id,
	})
}

func (s *APIServer) handleDeleteAccount(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (s *APIServer) handleTransfer(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

// apiFunc is a function type that represents an API handler.
type apiFunc func(http.ResponseWriter, *http.Request) error
type ApiError struct {
	Error string `json:"error"`
}

// makeHTTPHandleFunc converts an apiFunc to an http.HandlerFunc
// It takes an http.ResponseWriter and an *http.Request as parameters
// and returns an error if something goes wrong.
func makeHTTPHandleFunc(f apiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			writeJSON(w, http.StatusInternalServerError, ApiError{Error: err.Error()})
		}
	}
}
