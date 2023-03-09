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
	hostname string
	option_a string
	option_b string
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
	hostname, err := os.Hostname()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Printf("Hostname: %s", hostname)
	option_a := os.Getenv("OPTION_A")
	option_b := os.Getenv("OPTION_B")

	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = tmpl.Execute(w, struct {
		OptionA string
		OptionB string
		Host    string
		Vote    string
	}{
		OptionA: option_a,
		OptionB: option_b,
		Host:    hostname,
		Vote:    vote,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, cookie)
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
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// handle `/` route
	//fs := http.FileServer(http.Dir("./templates"))
	//http.Handle("/", fs)

	log.Fatal(http.ListenAndServe(":"+port, nil))

}
