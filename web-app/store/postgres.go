package store

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"

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
	DROP TABLE images
	`)

	_, err := s.db.Query(`
	CREATE TABLE IF NOT EXISTS images (imgname text, img bytea, classification json)
	`)
	if err != nil {
		return err
	}

	return nil
}

type Image struct {
	Img     string
	Classes []Class
}

type Classification struct {
	Images []struct {
		Classifiers []struct {
			Classes []Class
		}
	}
}

type Class struct {
	Class string
	Score float64
}

type rawImage struct {
	Img            []byte
	Classification string
}

func (s *Store) GetImages() ([]Image, error) {
	processedImages, err := s.getProcessedImages()

	if err != nil {
		return nil, nil
	}

	var images []Image
	for _, img := range processedImages {
		img, err := convertImage(img)
		if err != nil {
			return nil, err
		}
		images = append(images, img)
	}
	return images, nil
}

func (s *Store) getProcessedImages() ([]rawImage, error) {
	rows, err := s.db.Query("SELECT img, classification FROM images")
	if err != nil {
		return nil, err
	}

	var result []rawImage

	for rows.Next() {
		var img rawImage
		err := rows.Scan(&img.Img, &img.Classification)
		if err != nil {
			return nil, err
		}

		result = append(result, img)
	}

	return result, nil
}

func convertImage(img rawImage) (Image, error) {
	var classification Classification
	err := json.Unmarshal([]byte(img.Classification), &classification)
	if err != nil {
		return Image{}, err
	}

	var classes []Class
	for i := 0; i < 3; i++ {
		classes = append(classes, classification.Images[0].Classifiers[0].Classes[i])
	}

	return Image{
		Img:     base64.StdEncoding.EncodeToString(img.Img),
		Classes: classes,
	}, nil
}
