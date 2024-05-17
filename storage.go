package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/nrednav/cuid2"
)

type Storage interface {
	CreateAccount(*Account) (string, error)
	DeleteAccount(string) error
	UpdateAccount(*Account) error
	GetAccounts() ([]*Account, error)
	GetAccountById(string) (*Account, error)
	GetAccountByNumber(string) (*Account, error)
}

type PgStore struct {
	db *sql.DB
}

func NewPgStore() (*PgStore, error) {
	dbConnectionString := os.Getenv("DB_CONN_STRING")
	db, err := sql.Open("postgres", dbConnectionString)

	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &PgStore{db: db}, nil
}

func (s *PgStore) Init() error {

	return s.createAccountTable()

}

func (s *PgStore) createAccountTable() error {
	query := `CREATE TABLE IF NOT EXISTS accounts (
		id varchar(24) PRIMARY KEY,
		first_name text NOT NULL,
		last_name text NOT NULL,
		e_password text NOT NULL,
		number varchar(10) UNIQUE NOT NULL,
		balance float NOT NULL,
		created_at timestamp NOT NULL
		)`

	_, err := s.db.Exec(query)

	return err
}

func (s *PgStore) CreateAccount(a *Account) (string, error) {

	query := `INSERT INTO accounts (id, first_name, last_name, number, e_password, balance, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)  RETURNING id`

	id := cuid2.Generate()

	var insertedId string

	err := s.db.QueryRow(query, id, a.FirstName, a.LastName, a.Number, a.EPassword, a.Balance, a.CreatedAt).Scan(&insertedId)

	if err != nil {
		log.Fatalf("Failed to insert account: %v", err)
		return "", err
	}

	return insertedId, nil
}

func (s *PgStore) UpdateAccount(a *Account) error {
	query := `UPDATE accounts SET first_name = $2 , last_name = $3 WHERE id = $1`

	_, err := s.db.Query(query, a.Id, a.FirstName, a.LastName)

	return err

}

func (s *PgStore) DeleteAccount(id string) error {

	query := `DELETE FROM accounts WHERE id = $1`

	_, err := s.db.Query(query, id)

	return err
}

func (s *PgStore) GetAccountByNumber(number string) (*Account, error) {

	query := `SELECT * FROM accounts WHERE number = $1`

	rows, err := s.db.Query(query, number)

	if err != nil {
		return nil, err
	}

	for rows.Next() {
		return parseRow(rows)
	}

	return nil, fmt.Errorf("account with number %s not found", number)
}

func (s *PgStore) GetAccountById(id string) (*Account, error) {

	query := `SELECT * FROM accounts WHERE id = $1`

	rows, err := s.db.Query(query, id)

	if err != nil {
		return nil, err
	}

	for rows.Next() {
		return parseRow(rows)
	}

	return nil, fmt.Errorf("account with id %s not found", id)
}

func (s *PgStore) GetAccounts() ([]*Account, error) {

	query := `SELECT * FROM accounts`

	rows, err := s.db.Query(query)

	if err != nil {
		return nil, err
	}

	accounts := []*Account{}

	for rows.Next() {
		a, err := parseRow(rows)
		if err != nil {
			return nil, err
		}

		accounts = append(accounts, a)
	}

	return accounts, nil

}

func parseRow(r *sql.Rows) (*Account, error) {
	a := new(Account)

	if err := r.Scan(
		&a.Id,
		&a.FirstName,
		&a.LastName,
		&a.EPassword,
		&a.Number,
		&a.Balance,
		&a.CreatedAt); err != nil {
		return nil, err
	}

	return a, nil
}
