package main

import (
	"./aws"
	"./db"
	"encoding/json"
	"fmt"
	// "io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
)

const (
	imagesRoot = "/var/www/like-av.xyz/images/"
)

type Payload struct {
	Id         string
	Name       string
	File       string
	Img        string
	Similarity string
}

func UploadHandler(w http.ResponseWriter, r *http.Request) {
	// allow cross domain AJAX requests
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	checkPost(w, r)

	err := r.ParseMultipartForm(32 << 20) // maxMemory
	checkNil(w, err)

	file, handler, err := r.FormFile("upload")
	checkNil(w, err)
	defer file.Close()

	b, err := ioutil.ReadAll(file)
	checkNil(w, err)

	js, err := searchFace(b, handler.Filename)
	checkNil(w, err)

	w.Write(js)
}

func FeedbackHandler(w http.ResponseWriter, r *http.Request) {
	// allow cross domain AJAX requests
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	checkPost(w, r)

	decoder := json.NewDecoder(r.Body)
	val := map[string]string{}
	err := decoder.Decode(&val)
	checkNil(w, err)

	fmt.Println(val)
	if val["ox"] == "like" {
		b, err := ioutil.ReadFile(imagesRoot + val["id"] + "/" + val["file"])
		checkNil(w, err)
		checkNil(w, aws.InsertIndexFaceByImage(val["id"], b))
	}

	db.UpsertOneFeedback(val["id"], val["ox"], val["file"])
}

func main() {
	http.HandleFunc("/upload", UploadHandler)     // set router
	http.HandleFunc("/feedback", FeedbackHandler) // set router
	err := http.ListenAndServe(":9090", nil)      // set listen port
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func checkNil(w http.ResponseWriter, err error) {
	if err != nil {
		log.Fatal(err)
		// http.Error(*w, err.Error(), http.StatusInternalServerError)
		// return

		responseEmptyPayload(w)
	}
}

func checkPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		log.Fatal("Allowed POST method only")
		// http.Error(w, "Allowed POST method only", http.StatusMethodNotAllowed)
		// return

		responseEmptyPayload(w)
	}
}

func responseEmptyPayload(w http.ResponseWriter) {
	emptyPayload := Payload{
		Id: "",
	}

	js, _ := json.Marshal(emptyPayload)
	w.Write(js)
}

func searchFace(b []byte, fileName string) ([]byte, error) {
	result, err := aws.SearchFacesByImage(b)
	if err != nil {
		return nil, err
	}

	emptyPayload := Payload{
		Id:   "",
		File: fileName,
	}

	if result == nil {
		js, err := json.Marshal(emptyPayload)
		if err != nil {
			return nil, err
		}

		return js, nil
	} else {
		id := result.Id
		similarity := strconv.FormatFloat(result.Similarity, 'f', 2, 64)
		fmt.Println(id)
		fmt.Println(similarity)

		os.Mkdir(imagesRoot+id, os.ModePerm)
		err := ioutil.WriteFile(imagesRoot+id+"/"+fileName, b, 0644)
		if err != nil {
			return nil, err
		}

		val := db.FindOneActress(id)
		fmt.Println(val)

		if len(val) == 0 {
			js, err := json.Marshal(emptyPayload)
			if err != nil {
				return nil, err
			}

			return js, nil
		}

		payload := &Payload{
			Id:         id,
			Name:       val["name"],
			File:       fileName,
			Img:        val["img"],
			Similarity: similarity,
		}
		js, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}

		return js, nil
	}
}
