package repositories

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

type RepositoryAdapter struct {
	User *UserRepository
}

func NewRepositoryAdapter(path string) (*RepositoryAdapter, error) {
	var db, e = sql.Open("sqlite", path)
	if e != nil {
		return nil, e
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	var userRepo = NewUserRepository(db)
	e = userRepo.Init()
	if e != nil {
		return nil, e
	}

	return &RepositoryAdapter{User: userRepo}, nil
}
