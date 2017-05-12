package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/satori/go.uuid"

	"goji.io"
	"goji.io/pat"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// ErrorWithJSON Encodes errors into JSON for ease of use
func ErrorWithJSON(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	fmt.Fprintf(w, "{message: %q}", message)
}

// ResponseWithJSON Wraps responses in JSON for ease of use
func ResponseWithJSON(w http.ResponseWriter, json []byte, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	w.Write(json)
}

// Interruption Structure used
type Interruption struct {
	//	ID     int32  `json:"id"`
	UUID   string `json:"uuid"`
	What   string `json:"what"`
	When   int64  `json:"when"`
	Method int8   `json:"method"`
}

func main() {
	// Commandline Arguments:
	// -port=8080 -host localhost:27017 -pass 12345 -user myUser -db trailDB
	portPtr := flag.Int("port", 8080, "an int")
	userPtr := flag.String("user", "default", "Username")
	passPtr := flag.String("pass", "12345", "Password")
	mgoHostPtr := flag.String("host", "localhost:27017", "hostname:port")
	dbPtr := flag.String("db", "thejml-trail", "Database to use")
	testPtr := flag.Bool("test", false, "Validate Build")

	flag.Parse()

	port := fmt.Sprint(*portPtr)
	user := fmt.Sprint(*userPtr)
	pass := fmt.Sprint(*passPtr)
	db := fmt.Sprint(*dbPtr)
	mgoHost := fmt.Sprint(*mgoHostPtr)
	test := *testPtr

	// Better info should be put here.
	if test != false {
		log.Println("Test Passed")
		return
	}

	session, err := mgo.Dial("mongodb://" + user + ":" + pass + "@" + mgoHost + "/" + db)
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
	// curl localhost:8080/int
	mux.HandleFunc(pat.Get("/int"), allInterruptions(session))

	// Update an existing Interruption by ID
	mux.HandleFunc(pat.Put("/int/:searchuuid"), updateInterruption(session))

	// Search Interruptions
	// curl localhost:8080/int/:uuid
	mux.HandleFunc(pat.Get("/int/:searchuuid"), searchInterruptions(session))

	// Delete a interruption by ID
	mux.HandleFunc(pat.Delete("/int/:searchuuid"), deleteInterruption(session))
	log.Println("Server up and running on localhost:" + port)
	log.Fatal(http.ListenAndServe("localhost:"+port, mux))
}

func ensureInterruptionIndex(s *mgo.Session) {
	session := s.Copy()
	defer session.Close()

	c := session.DB("thejml-trail").C("interruptions")

	index := mgo.Index{
		Key:        []string{"uuid"},
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
		u1 := uuid.NewV4()
		t := time.Now().UTC().Unix()

		var interruption Interruption
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&interruption)
		if err != nil {
			ErrorWithJSON(w, "Incorrect body", http.StatusBadRequest)
			log.Println("Failed insert interruption: ", err)
			return
		}
		interruption.When = t
		interruption.UUID = fmt.Sprintf("%s", u1)
		c := session.DB("thejml-trail").C("interruptions")

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

		c := session.DB("thejml-trail").C("interruptions")

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

func searchInterruptions(s *mgo.Session) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		session := s.Copy()
		defer session.Close()

		searchuuid := pat.Param(r, "searchuuid")

		c := session.DB("thejml-trail").C("interruptions")

		var interruptions []Interruption
		err := c.Find(bson.M{"uuid": searchuuid}).All(&interruptions)
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

// Overwites anything it doesn't get in the posted object with blanks.
func updateInterruption(s *mgo.Session) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		session := s.Copy()
		defer session.Close()

		searchuuid := pat.Param(r, "searchuuid")

		var interruption Interruption
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&interruption)
		if err != nil {
			ErrorWithJSON(w, "Incorrect body", http.StatusBadRequest)
			return
		}

		c := session.DB("thejml-trail").C("interruptions")

		err = c.Update(bson.M{"uuid": searchuuid}, &interruption)
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

		searchuuid := pat.Param(r, "searchuuid")

		c := session.DB("thejml-trail").C("interruptions")

		err := c.Remove(bson.M{"uuid": searchuuid})
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
