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
	// temp
	s.db.Query(`
	DROP TABLE text
	`)

	_, err := s.db.Query(`
	CREATE TABLE IF NOT EXISTS text(
		text varchar(255) PRIMARY KEY
	)`)
	if err != nil {
		return err
	}

	// temp
	_, err = s.db.Query(`
	INSERT INTO text VALUES ('dummydata')
	`)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) GetProcessedText() ([]string, error) {
	rows, err := s.db.Query("SELECT * FROM text")
	if err != nil {
		return nil, err
	}

	var result []string

	for rows.Next() {
		var text string
		err := rows.Scan(&text)
		if err != nil {
			return nil, err
		}

		result = append(result, text)
	}

	return result, nil
}
