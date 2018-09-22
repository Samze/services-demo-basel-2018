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
	// temp drop everything while for testing
	s.db.Query(`
	DROP TABLE text
	`)

	s.db.Query(`
	DROP TABLE images
	`)

	_, err := s.db.Query(`
	CREATE TABLE IF NOT EXISTS images (imgname text, img bytea)
	`)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) GetProcessedImages() ([][]byte, error) {
	rows, err := s.db.Query("SELECT img FROM images")
	if err != nil {
		return nil, err
	}

	var result [][]byte

	for rows.Next() {
		var img []byte
		err := rows.Scan(&img)
		if err != nil {
			return nil, err
		}

		result = append(result, img)
	}

	return result, nil
}
