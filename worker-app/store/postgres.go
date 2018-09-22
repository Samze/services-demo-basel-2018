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
	CREATE TABLE IF NOT EXISTS text(
		text varchar(255) PRIMARY KEY
	)`)

	if err != nil {
		return err
	}

	_, err = s.db.Query(`CREATE TABLE IF NOT EXISTS images (imgname text, img bytea)`)
	return err
}

func (s *Store) AddText(text string) error {
	sqlStatement := "INSERT INTO text VALUES ($1)"
	_, err := s.db.Exec(sqlStatement, text)
	return err
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
