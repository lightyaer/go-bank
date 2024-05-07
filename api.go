package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type APIServer struct {
	listenAddr string
	store      Storage
}

func NewAPIServer(listenAddr string, store Storage) *APIServer {
	return &APIServer{
		listenAddr: listenAddr,
		store:      store,
	}
}

func (s *APIServer) Run() {
	router := mux.NewRouter()

	// router.Use(middleware)

	router.HandleFunc("/account", createHTTPHandler(s.handleAccounts))
	router.HandleFunc("/account/{id}", createHTTPHandler(s.handleAccount))
	router.HandleFunc("/transfer", createHTTPHandler(s.handleTransferAccount))

	log.Println("JSON api server is running on port: ", s.listenAddr)

	http.ListenAndServe(s.listenAddr, router)
}

func (s *APIServer) handleAccount(w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case http.MethodGet:
		{
			return s.handleGetAccount(w, r)
		}
	case http.MethodDelete:
		{
			return s.handleDeleteAccount(w, r)
		}
	case http.MethodPatch:
		{
			return s.handleTransferAccount(w, r)
		}
	}

	return fmt.Errorf("method not allowed %s", r.Method)

}

func (s *APIServer) handleAccounts(w http.ResponseWriter, r *http.Request) error {

	switch r.Method {
	case http.MethodGet:
		{
			return s.handleGetAccounts(w, r)
		}
	case http.MethodPost:
		{
			return s.handleCreateAccount(w, r)
		}
	}

	return fmt.Errorf("method not allowed %s", r.Method)

}

func (s *APIServer) handleGetAccounts(w http.ResponseWriter, _r *http.Request) error {

	accounts, err := s.store.GetAccounts()

	if err != nil {
		return err
	}

	return WriteJSON(w, http.StatusOK, accounts)
}

func (s *APIServer) handleGetAccount(w http.ResponseWriter, r *http.Request) error {

	id, err := parseId(r)
	if err != nil {
		return err
	}

	if account, err := s.store.GetAccountById(int(id)); err != nil {
		return WriteJSON(w, http.StatusOK, &account)
	} else {
		return err
	}

}

func (s *APIServer) handleCreateAccount(w http.ResponseWriter, r *http.Request) error {

	accountRequest := CreateAccountRequest{}
	if err := json.NewDecoder(r.Body).Decode(&accountRequest); err != nil {
		return err
	}

	account := NewAccount(accountRequest.FirstName, accountRequest.LastName)

	if insertedId, err := s.store.CreateAccount(account); err != nil {
		return err
	} else {
		return WriteJSON(w, http.StatusOK, map[string]string{"created": insertedId})
	}

}

func (s *APIServer) handleDeleteAccount(w http.ResponseWriter, r *http.Request) error {
	id, err := parseId(r)
	if err != nil {
		return err
	}

	if err := s.store.DeleteAccount(int(id)); err != nil {
		return err
	}
	return WriteJSON(w, http.StatusOK, map[string]int{"deleted": id})

}
func (s *APIServer) handleTransferAccount(w http.ResponseWriter, r *http.Request) error {

	return nil

}

func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)

	return json.NewEncoder(w).Encode(v)
}

type apiFunc func(http.ResponseWriter, *http.Request) error

type ApiError struct {
	Error string
}

func createHTTPHandler(f apiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			// handle error
			WriteJSON(w, http.StatusBadRequest, ApiError{Error: err.Error()})
		}
	}
}

func middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

func parseId(r *http.Request) (int, error) {
	idStr := mux.Vars(r)["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return id, fmt.Errorf("invalid id given %s", idStr)
	}
	return id, nil
}
