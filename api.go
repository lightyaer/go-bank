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

	router.HandleFunc("/login", createHTTPHandler(s.handleLogin))

	router.HandleFunc("/account", createHTTPHandler(s.handleAccounts))
	router.HandleFunc("/account/{id}", withJWTAuth(createHTTPHandler(s.handleAccount)))
	router.HandleFunc("/transfer", createHTTPHandler(s.handleTransferAccount))

	log.Println("JSON api server is running on port: ", s.listenAddr)

	http.ListenAndServe(s.listenAddr, router)
}

func (s *APIServer) handleLogin(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return fmt.Errorf("method not allowed %s", r.Method)
	}

	var req LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return err
	}

	acc, err := s.store.GetAccountByNumber(string(req.Number))

	if err != nil {
		return err
	}

	if !acc.ValidPassword(req.Password) {
		return fmt.Errorf("invalid password")
	}

	token, err := createJWT(acc)
	if err != nil {
		return err
	}

	response := LoginResponse{
		Token:  token,
		Number: acc.Number,
	}

	return WriteJSON(w, http.StatusOK, response)
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
			return s.handleGetAccounts(w)
		}
	case http.MethodPost:
		{
			return s.handleCreateAccount(w, r)
		}
	}

	return fmt.Errorf("method not allowed %s", r.Method)

}

func (s *APIServer) handleGetAccounts(w http.ResponseWriter) error {

	accounts, err := s.store.GetAccounts()

	if err != nil {
		return err
	}

	return WriteJSON(w, http.StatusOK, accounts)
}

func (s *APIServer) handleGetAccount(w http.ResponseWriter, r *http.Request) error {

	accountId := r.Context().Value(keyAccount).(AuthRequestContext).Id

	fmt.Println("account id: ", accountId)

	if account, err := s.store.GetAccountById(accountId); err != nil {
		return err
	} else {
		fmt.Printf("%v", account)
		return WriteJSON(w, http.StatusOK, account)

	}

}

func (s *APIServer) handleCreateAccount(w http.ResponseWriter, r *http.Request) error {

	accountRequest := CreateAccountRequest{}
	if err := json.NewDecoder(r.Body).Decode(&accountRequest); err != nil {
		return err
	}

	account, err := NewAccount(accountRequest.FirstName, accountRequest.LastName, accountRequest.Password)

	if err != nil {
		return err
	}

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

func withJWTAuth(handlerFunc http.HandlerFunc) http.HandlerFunc {
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

		*r = *r.WithContext(context.WithValue(r.Context(), keyAccount, AuthRequestContext{Number: accountNumber, Id: accountId}))

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
