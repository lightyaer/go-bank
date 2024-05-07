package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
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
	router.HandleFunc("/account/{id}", withJWTAuth(createHTTPHandler(s.handleAccount), s.store))
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

	id := mux.Vars(r)["id"]

	if account, err := s.store.GetAccountById(id); err != nil {
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

	insertedId, err := s.store.CreateAccount(account)

	if err != nil {
		return err
	}

	account.Id = insertedId

	tokenString, err := createJWT(account)

	if err != nil {
		return err
	}

	fmt.Println("token: ", tokenString)

	w.Header().Add("Set-Cookie", `gb_session=`+tokenString)

	return WriteJSON(w, http.StatusOK, map[string]string{"created": insertedId})

}

func (s *APIServer) handleDeleteAccount(w http.ResponseWriter, r *http.Request) error {
	id := mux.Vars(r)["id"]

	if err := s.store.DeleteAccount(id); err != nil {
		return err
	}
	return WriteJSON(w, http.StatusOK, map[string]string{"deleted": id})

}
func (s *APIServer) handleTransferAccount(w http.ResponseWriter, r *http.Request) error {

	return nil

}

func withJWTAuth(handlerFunc http.HandlerFunc, store Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("gb_session")

		if err != nil {
			WriteJSON(w, http.StatusUnauthorized, ApiError{Error: "Access Denied"})
			return
		}

		tokenStr := cookie.Value

		token, err := validateJWT(tokenStr)

		if err != nil || !token.Valid {
			WriteJSON(w, http.StatusUnauthorized, ApiError{Error: "Access Denied"})
			return
		}

		claims := token.Claims.(jwt.MapClaims)

		accountId := claims["accountId"].(string)
		accountNumber := claims["accountNumber"].(string)

		id := mux.Vars(r)["id"]

		if id != accountId {
			WriteJSON(w, http.StatusUnauthorized, ApiError{Error: "Access Denied"})
			return
		}

		ctx := r.Context()
		*r = *r.WithContext(context.WithValue(ctx, "account", map[string]string{accountId: accountId, accountNumber: accountNumber}))

		handlerFunc(w, r)

	}
}

func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)

	return json.NewEncoder(w).Encode(v)
}

type apiFunc func(http.ResponseWriter, *http.Request) error

type ApiError struct {
	Error string `json:"error"`
}

func createHTTPHandler(f apiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			// handle error
			WriteJSON(w, http.StatusBadRequest, ApiError{Error: err.Error()})
		}
	}
}

func createJWT(account *Account) (string, error) {
	claims := &jwt.MapClaims{
		"expiresAt":     jwt.NewNumericDate(time.Now().Add(time.Minute * 2)),
		"issuer":        "gobank",
		"issuesAt":      jwt.NewNumericDate(time.Now()),
		"accountNumber": account.Number,
		"accountId":     account.Id,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret := []byte(os.Getenv("JWT_SECRET"))

	return token.SignedString(secret)
}

func validateJWT(token string) (*jwt.Token, error) {
	secret := os.Getenv("JWT_SECRET")

	return jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(secret), nil
	})
}
