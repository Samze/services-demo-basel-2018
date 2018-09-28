package store

import (
	"database/sql"

	_ "github.com/lib/pq"
)

type Store struct {
	db *sql.DB
}

func NewStore(conn string) (*Store, error) {
	s := Store{}
	db, err := sql.Open("postgres", conn)
	if err != nil {
		return nil, err
	}
	s.db = db

	err = s.createTables()
	if err != nil {
		return nil, err
	}

	return &s, nil
}

func (s *Store) createTables() error {
	_, err := s.db.Query(`
	CREATE TABLE IF NOT EXISTS images (imgname text, img bytea, classification json)
	`)
	return err
}

func (s *Store) AddImage(img, classification []byte) error {
	sqlStatement := `INSERT INTO images VALUES ($1, $2, $3)`
	_, err := s.db.Exec(sqlStatement, "imgname", img, string(classification))
	return err
}
