package main

import (
	"encoding/json"
	"fmt"
	"flag"
	"log"
	"net/http"
	"time"

	"goji.io"
	"goji.io/pat"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func ErrorWithJSON(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	fmt.Fprintf(w, "{message: %q}", message)
}

func ResponseWithJSON(w http.ResponseWriter, json []byte, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	w.Write(json)
}

type Interruption struct {
	ID       int32    `json:"id"`
	What     string   `json:"what"`
	When     int64    `json:"when"`
	Method   int8     `json:"method"`
}

func main() {
	// Commandline Arguments:
	// -port=8080 -host localhost:27017 -pass 12345 -user myUser -db trailDB
	portPtr    := flag.Int("port", 8080, "an int")
	userPtr    := flag.String("user", "default", "Username")
	passPtr    := flag.String("pass", "12345", "Password")
	mgoHostPtr := flag.String("host", "localhost:27017", "hostname:port")
        dbPtr      := flag.String("db", "thejml-trail", "Database to use")

	flag.Parse()

	port    := fmt.Sprint(*portPtr)
	user    := fmt.Sprint(*userPtr)
	pass    := fmt.Sprint(*passPtr)
	db	:= fmt.Sprint(*dbPtr)
	mgoHost := fmt.Sprint(*mgoHostPtr)


	session, err := mgo.Dial("mongodb://"+user+":"+pass+"@"+mgoHost+"/"+db)
	if err != nil {
		panic(err)
	}
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	ensureInterruptionIndex(session)

	mux := goji.NewMux()
	// Add new ToDo
	// curl localhost:8080/interruptions -X POST -H "Content-Type: application/json" -d '{"id":0,"what":"Finish making interruption API","method":0}'
	mux.HandleFunc(pat.Post("/int"), addInterruption(session))

	// List all current Interruptions
	// curl localhost:8080/interruptions
	mux.HandleFunc(pat.Get("/int"), allInterruptions(session))

	// Update an existing Interruption by ID
	mux.HandleFunc(pat.Put("/int/:id"), updateInterruption(session))

	// Delete a interruption by ID
	mux.HandleFunc(pat.Delete("/int/:id"), deleteInterruption(session))
	http.ListenAndServe("localhost:"+port, mux)
}

func ensureInterruptionIndex(s *mgo.Session) {
	session := s.Copy()
	defer session.Close()

	c := session.DB("thejml-trail").C("interruptions")

	index := mgo.Index{
		Key:        []string{"id"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
	}
	err := c.EnsureIndex(index)
	if err != nil {
		panic(err)
	}
}

func addInterruption(s *mgo.Session) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		session := s.Copy()
		defer session.Close()
		t := time.Now().UTC().Unix();

		var interruption Interruption
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&interruption)
		if err != nil {
			ErrorWithJSON(w, "Incorrect body", http.StatusBadRequest)
			return
		}
		interruption.When=t;
		c := session.DB("thejml-trail").C("iterruptions")

		err = c.Insert(interruption)
		if err != nil {
			if mgo.IsDup(err) {
				ErrorWithJSON(w, "Interruption with this ID already exists", http.StatusBadRequest)
				return
			}

			ErrorWithJSON(w, "Database error", http.StatusInternalServerError)
			log.Println("Failed insert interruption: ", err)
			return
		}

		// Respond and redirect to the resulting interruption
		// w.Header().Set("Content-Type", "application/json")
		// w.Header().Set("Location", r.URL.Path+"/"+interruption.ID)
		// w.WriteHeader(http.StatusCreated)
	}
}

func allInterruptions(s *mgo.Session) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		session := s.Copy()
		defer session.Close()

		c := session.DB("thejml-trail").C("iterruptions")

		var interruptions []Interruption
		err := c.Find(bson.M{}).All(&interruptions)
		if err != nil {
			ErrorWithJSON(w, "Database error", http.StatusInternalServerError)
			log.Println("Failed get all books: ", err)
			return
		}

		respBody, err := json.MarshalIndent(interruptions, "", "  ")
		if err != nil {
			log.Fatal(err)
		}

		ResponseWithJSON(w, respBody, http.StatusOK)
	}
}

func updateInterruption(s *mgo.Session) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		session := s.Copy()
		defer session.Close()

		id := pat.Param(r, "id")

		var interruption Interruption
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&interruption)
		if err != nil {
			ErrorWithJSON(w, "Incorrect body", http.StatusBadRequest)
			return
		}

		c := session.DB("thejml-trail").C("iterruptions")

		err = c.Update(bson.M{"id": id}, &interruption)
		if err != nil {
			switch err {
			default:
				ErrorWithJSON(w, "Database error", http.StatusInternalServerError)
				log.Println("Failed update book: ", err)
				return
			case mgo.ErrNotFound:
				ErrorWithJSON(w, "Interruption not found", http.StatusNotFound)
				return
			}
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func deleteInterruption(s *mgo.Session) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		session := s.Copy()
		defer session.Close()

		id := pat.Param(r, "id")

		c := session.DB("thejml-trail").C("iterruptions")

		err := c.Remove(bson.M{"id": id})
		if err != nil {
			switch err {
			default:
				ErrorWithJSON(w, "Database error", http.StatusInternalServerError)
				log.Println("Failed to delete interruption: ", err)
				return
			case mgo.ErrNotFound:
				ErrorWithJSON(w, "Interruption not found", http.StatusNotFound)
				return
			}
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
