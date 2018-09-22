package store

import (
	"database/sql"
	"io/ioutil"

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
	CREATE TABLE IF NOT EXISTS text(
		text varchar(255) PRIMARY KEY
	)`)
	if err != nil {
		return err
	}

	_, err = s.db.Query(`CREATE TABLE IF NOT EXISTS images (imgname text, img bytea)`)
	if err != nil {
		return err
	}

	// temp some random data
	_, err = s.db.Query(`
	INSERT INTO text VALUES ('dummydata')
	`)
	if err != nil {
		return err
	}

	img, err := ioutil.ReadFile("osbapi.png")
	if err != nil {
		return err
	}

	sqlStatement := "INSERT INTO images VALUES ($1, $2)"
	_, err = s.db.Exec(sqlStatement, "test", img)
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
