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
	CREATE TABLE IF NOT EXISTS images (imgname text, img bytea, classification json)
	`)
	if err != nil {
		return err
	}

	sqlStatement := `INSERT INTO images VALUES ($1, $2, $3)`

	img, err := ioutil.ReadFile("osbapi.png")
	if err != nil {
		return err
	}

	_, err = s.db.Exec(sqlStatement, "imgname", img, `{
	"images": [{
		"classifiers": [{
			"classifier_id": "default",
			"name": "default",
			"classes": [{
				"class": "banana",
				"score": 0.562,
				"type_hierarchy": "/fruit/banana"
			}, {
				"class": "fruit",
				"score": 0.788
			}, {
				"class": "diet (food)",
				"score": 0.528,
				"type_hierarchy": "/food/diet (food)"
			}, {
				"class": "food",
				"score": 0.528
			}, {
				"class": "honeydew",
				"score": 0.5,
				"type_hierarchy": "/fruit/melon/honeydew"
			}, {
				"class": "melon",
				"score": 0.501
			}, {
				"class": "olive color",
				"score": 0.973
			}, {
				"class": "lemon yellow color",
				"score": 0.789
			}]
		}],
		"image": "fruitbowl.jpg"
	}],
	"images_processed": 1,
	"custom_classes": 0
}`)

	if err != nil {
		return err
	}

	return nil
}

type Image struct {
	Img            []byte
	Classification string
}

func (s *Store) GetProcessedImages() ([]Image, error) {
	rows, err := s.db.Query("SELECT img, classification FROM images")
	if err != nil {
		return nil, err
	}

	var result []Image

	for rows.Next() {
		var img Image
		err := rows.Scan(&img.Img, &img.Classification)
		if err != nil {
			return nil, err
		}

		result = append(result, img)
	}

	return result, nil
}

func (s *Store) GetClassifications() ([][]byte, error) {
	rows, err := s.db.Query("SELECT classification FROM images")
	if err != nil {
		return nil, err
	}

	var result [][]byte

	for rows.Next() {
		var class []byte
		err := rows.Scan(&class)
		if err != nil {
			return nil, err
		}

		result = append(result, class)
	}

	return result, nil
}
