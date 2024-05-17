package main

import (
	"time"

	nanoid "github.com/matoous/go-nanoid/v2"
	"golang.org/x/crypto/bcrypt"
)

type key string

const (
	keyAccount key = "account"
)

type AuthRequestContext struct {
	Number string
	Id     string
}

type LoginResponse struct {
	Number string `json:"number"`
	Token  string `json:"token"`
}

type LoginRequest struct {
	Number   string `json:"number"`
	Password string `json:"password"`
}

type TransferRequest struct {
	ToAccount string  `json:"toAccount"`
	Amount    float64 `json:"amount"`
}

type CreateAccountRequest struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Password  string `json:"password"`
}

type Account struct {
	Id        string    `json:"id"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Number    string    `json:"number"`
	EPassword string    `json:"-"`
	Balance   float64   `json:"balance"`
	CreatedAt time.Time `json:"createdAt"`
}

func (a *Account) ValidPassword(pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(a.EPassword), []byte(pw)) == nil
}

func NewAccount(firstName, lastName, password string) (*Account, error) {
	enPw, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	accountNumber, err := nanoid.New(10)

	if err != nil {
		return nil, err
	}

	return &Account{
		FirstName: firstName,
		LastName:  lastName,
		EPassword: string(enPw),
		Number:    accountNumber,
		Balance:   0,
		CreatedAt: time.Now().UTC(),
	}, nil
}
