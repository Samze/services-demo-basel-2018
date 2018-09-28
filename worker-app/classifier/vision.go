package classifier

import (
	"bytes"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
)

type Vision struct {
	url    *url.URL
	apiKey string
	client *http.Client
}

func NewVision(uri, apiKey string) (*Vision, error) {
	client := &http.Client{}
	v := Vision{apiKey: apiKey, client: client}

	u, err := url.Parse(uri)
	if err != nil {
		return &v, err
	}
	q := u.Query()
	// Hardcore the latest service version since it's required param
	q.Add("version", "2018-03-19")
	u.RawQuery = q.Encode()
	v.url = u

	return &v, nil
}

func (v *Vision) ClassifyImage(img []byte) (classification []byte, err error) {
	buf := new(bytes.Buffer)
	w := multipart.NewWriter(buf)
	// TODO provide real name
	image, err := w.CreateFormFile("images_file", "fruitbowl.jpg")
	if err != nil {
		return classification, err
	}

	_, err = image.Write(img)
	if err != nil {
		return classification, err
	}

	err = w.Close()
	if err != nil {
		return classification, err
	}

	url := *v.url
	url.Path = path.Join(v.url.Path, "v3/classify")
	req, err := http.NewRequest("POST", url.String(), buf)
	if err != nil {
		return classification, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.SetBasicAuth("apikey", v.apiKey)
	res, err := v.client.Do(req)
	if err != nil {
		return classification, err
	}

	classification, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return classification, err
	}

	return classification, nil
}

func (v *Vision) DetectFaces(img []byte) {
	// TODO implement
}
