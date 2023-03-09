package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
)

type vars struct {
	Hostname string
	Option_a string
	Option_b string
}

type voteData struct {
	VoterID string `json:"voter_id"`
	Vote    string `json:"vote"`
}

func getRedis(ctx context.Context) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	return rdb.WithContext(ctx)
}

func hello(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	cookie, err := r.Cookie("voter_id")
	if err == http.ErrNoCookie {
		rand.Seed(time.Now().UnixNano())
		cookie = &http.Cookie{
			Name:  "voter_id",
			Value: strconv.FormatInt(rand.Int63(), 16),
		}
	}
	if err != nil {
		log.Println(err)
	}
	vote := ""

	if r.Method == http.MethodPost {
		err := r.ParseForm()
		if err != nil {
			log.Println(err)
		}
		vote = r.FormValue("vote")
		log.Printf("Received vote for %s", vote)
		data := voteData{
			VoterID: cookie.Value,
			Vote:    vote,
		}
		rdb := getRedis(ctx)
		jsonData, err := json.Marshal(data)
		if err != nil {
			log.Println(err)
		}
		err = rdb.RPush(ctx, "votes", jsonData).Err()
		if err != nil {
			log.Println(err)
		}
	}
	Hostname, err := os.Hostname()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Printf("Hostname: %s", Hostname)
	Option_a := os.Getenv("OPTION_A")
	Option_b := os.Getenv("OPTION_B")

	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = tmpl.Execute(w, struct {
		Option_a string
		Option_b string
		Hostname string
		Vote     string
	}{
		Option_a: Option_a,
		Option_b: Option_b,
		Hostname: Hostname,
		Vote:     vote,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Remove the following line to fix the error
	//http.ResponseWriter.WriteHeader(http.StatusOK)
}

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Panicln("Error loading environment variables", err)
	}
	fmt.Println("Port", os.Getenv("PORT"))
	port := os.Getenv("PORT")

	ctx := context.Background()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		hello(w, r, ctx)
	})
	fs := http.FileServer(http.Dir("./templates/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// handle `/` route
	//fs := http.FileServer(http.Dir("./templates"))
	//http.Handle("/", fs)

	log.Fatal(http.ListenAndServe(":"+port, nil))

}
