package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"

	"github.com/Samze/services-demo-basel-2018/web-app/queue"
	"github.com/Samze/services-demo-basel-2018/web-app/store"
)

type Storer interface {
	GetImages() ([]store.Image, error)
}

type Queuer interface {
	PublishImage([]byte) error
	Destroy()
}

const (
	port                 = "8080"
	googleAppCredentials = "GOOGLE_APPLICATION_CREDENTIALS"
)

func main() {
	c, err := NewConfig()
	if err != nil {
		log.Fatalf("could not load config %+v", err)
	}
	defer c.RemoveTmpFile()

	queue, err := queue.NewQueue(c.ProjectID, c.TopicID)
	if err != nil {
		log.Fatalf("could not load config %+v", err)
	}

	defer queue.Destroy()

	store, err := store.NewStore(c.ConnectionString)
	if err != nil {
		log.Fatalf("Could not connect to store %+v", err)
	}

	http.HandleFunc("/images", postImageHandler(queue))
	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("./css"))))
	http.HandleFunc("/", getHandler(store))

	fmt.Println("Listening on port:", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", port), nil))
}

func postImageHandler(q Queuer) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		file, _, err := r.FormFile("image")
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to read img: %v", err), http.StatusInternalServerError)
			return
		}
		defer file.Close()

		imgBuf := bytes.NewBuffer(nil)
		if _, err := io.Copy(imgBuf, file); err != nil {
			http.Error(w, fmt.Sprintf("Failed to copy img: %v", err), http.StatusInternalServerError)
			return
		}

		if err := q.PublishImage(imgBuf.Bytes()); err != nil {
			http.Error(w, fmt.Sprintf("Failed to publish img: %v", err), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func getHandler(s Storer) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		t, err := template.ParseFiles("tmpl/home.html")
		if err != nil {
			fmt.Fprintf(w, "err getting template %+v", err)
		}

		images, err := s.GetImages()
		if err != nil {
			fmt.Fprintf(w, "err getting images %+v", err)
		}

		data := map[string][]store.Image{
			"Images": images,
		}

		t.Execute(w, data)
	}
}
