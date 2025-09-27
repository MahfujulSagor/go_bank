package main

import (
	"encoding/json"
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
	router.HandleFunc("/transfer", makeHTTPHandleFunc(s.handleTransfer))

	log.Println("Starting server on:", s.Addr)
	http.ListenAndServe(s.Addr, router)
}

func (s *APIServer) handleAccount(w http.ResponseWriter, r *http.Request) error {
	if r.Method == http.MethodPost {
		return s.handleCreateAccount(w, r)
	}
	if r.Method == http.MethodGet {
		return s.handleGetAccount(w, r)
	}

	return writeJSON(w, http.StatusBadRequest, ApiError{Error: "method not allowed"})
}

// handleGetAccount handles the GET /account request.
// It retrieves all accounts from the storage and returns them as JSON.
// Limited to 10 accounts for simplicity.
func (s *APIServer) handleGetAccount(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet {
		return writeJSON(w, http.StatusBadRequest, ApiError{Error: "method not allowed"})
	}
	accounts, err := s.store.GetAccounts()
	if err != nil {
		return writeJSON(w, http.StatusInternalServerError, ApiError{Error: "failed to get accounts"})
	}
	if len(accounts) == 0 {
		return writeJSON(w, http.StatusNotFound, ApiError{Error: "no accounts found"})
	}

	return writeJSON(w, http.StatusOK, accounts)
}

func (s *APIServer) handleGetAccountByID(w http.ResponseWriter, r *http.Request) error {
	if r.Method == http.MethodGet {
		if r.Method != http.MethodGet {
			return writeJSON(w, http.StatusBadRequest, ApiError{Error: "method not allowed"})
		}

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

	if r.Method == http.MethodDelete {
		return s.handleDeleteAccount(w, r)
	}
	if r.Method == http.MethodPut {
		return s.handleUpdateAccountBalance(w, r)
	}

	return writeJSON(w, http.StatusBadRequest, ApiError{Error: "method not allowed"})
}

// handleCreateAccount handles the POST /account request.
func (s *APIServer) handleCreateAccount(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return writeJSON(w, http.StatusBadRequest, ApiError{Error: "method not allowed"})
	}

	createAccountReq := &CreateAccountRequest{}
	if err := json.NewDecoder(r.Body).Decode(createAccountReq); err != nil {
		return writeJSON(w, http.StatusBadRequest, ApiError{Error: "invalid request payload"})
	}
	defer r.Body.Close()

	if createAccountReq.FirstName == "" || createAccountReq.LastName == "" {
		return writeJSON(w, http.StatusBadRequest, ApiError{Error: "first name and last name are required"})
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
	if r.Method != http.MethodDelete {
		return writeJSON(w, http.StatusBadRequest, ApiError{Error: "method not allowed"})
	}

	idStr := mux.Vars(r)["id"]
	if idStr == "" {
		return writeJSON(w, http.StatusBadRequest, ApiError{Error: "missing account id"})
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return writeJSON(w, http.StatusBadRequest, ApiError{Error: "invalid account id"})
	}

	deletedID, err := s.store.DeleteAccount(id)
	if err != nil {
		return writeJSON(w, http.StatusInternalServerError, ApiError{Error: "failed to delete account"})
	}
	if deletedID == 0 {
		return writeJSON(w, http.StatusNotFound, ApiError{Error: "account not found"})
	}

	return writeJSON(w, http.StatusOK, map[string]int64{
		"id": deletedID,
	})
}

func (s *APIServer) handleUpdateAccountBalance(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPut {
		return writeJSON(w, http.StatusBadRequest, ApiError{Error: "method not allowed"})
	}

	idStr := mux.Vars(r)["id"]
	if idStr == "" {
		return writeJSON(w, http.StatusBadRequest, ApiError{Error: "missing account id"})
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return writeJSON(w, http.StatusBadRequest, ApiError{Error: "invalid account id"})
	}

	updateAccountReq := &UpdateAccountBalanceRequest{}
	if err := json.NewDecoder(r.Body).Decode(updateAccountReq); err != nil {
		return writeJSON(w, http.StatusBadRequest, ApiError{Error: "invalid request payload"})
	}
	defer r.Body.Close()

	if updateAccountReq.Balance < 0 {
		return writeJSON(w, http.StatusBadRequest, ApiError{Error: "balance cannot be negative"})
	}
	if updateAccountReq.Number <= 0 {
		return writeJSON(w, http.StatusBadRequest, ApiError{Error: "invalid account number"})
	}

	// Check if account exists
	existingAccount, err := s.store.GetAccountByID(id)
	if err != nil {
		return writeJSON(w, http.StatusInternalServerError, ApiError{Error: "failed to get account"})
	}
	if existingAccount == nil {
		return writeJSON(w, http.StatusNotFound, ApiError{Error: "account not found"})
	}

	account := &Account{
		ID:      id,
		Balance: updateAccountReq.Balance,
		Number:  updateAccountReq.Number,
	}

	updatedID, err := s.store.UpdateAccountBalance(id, account)
	if err != nil {
		return writeJSON(w, http.StatusInternalServerError, ApiError{Error: "failed to update account"})
	}
	if updatedID == 0 {
		return writeJSON(w, http.StatusNotFound, ApiError{Error: "account not found"})
	}

	return writeJSON(w, http.StatusOK, map[string]int64{
		"id": updatedID,
	})
}
func (s *APIServer) handleTransfer(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return writeJSON(w, http.StatusBadRequest, ApiError{Error: "method not allowed"})
	}

	transferReq := &TransferRequest{}
	if err := json.NewDecoder(r.Body).Decode(transferReq); err != nil {
		return writeJSON(w, http.StatusBadRequest, ApiError{Error: "invalid request payload"})
	}
	defer r.Body.Close()

	if transferReq.FromAccountNo == transferReq.ToAccountNo {
		return writeJSON(w, http.StatusBadRequest, ApiError{Error: "cannot transfer to the same account"})
	}

	if transferReq.Amount <= 0 {
		return writeJSON(w, http.StatusBadRequest, ApiError{Error: "amount must be greater than zero"})
	}

	if err := s.store.TransferMoney(transferReq.FromAccountNo, transferReq.ToAccountNo, transferReq.Amount); err != nil {
		return writeJSON(w, http.StatusInternalServerError, ApiError{Error: "failed to transfer funds"})
	}
	return writeJSON(w, http.StatusOK, map[string]string{
		"status": "transfer successful",
	})
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
