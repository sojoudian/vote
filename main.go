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
	"path/filepath"
	"strconv"
	"time"
)

type vars struct {
	hostname string
}

type voteData struct {
	VoterID string `json:"voter_id"`
	Vote    string `json:"vote"`
}

func getRedis(ctx context.Context) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
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

	optionA := os.Getenv("OPTION_A")
	optionB := os.Getenv("OPTION_B")

	templateDir := filepath.Join("templates", "index.html")
	tmpl := template.Must(template.ParseFiles(templateDir))
	tmpl = tmpl.Funcs(template.FuncMap{"option_a": optionA})
	tmpl = tmpl.Funcs(template.FuncMap{"option_b": optionB})
	err = tmpl.Execute(w, struct {
		OptionA string
		OptionB string
		Host    string
		Vote    string
	}{
		OptionA: optionA,
		OptionB: optionB,
		Host:    hostname,
		Vote:    vote,
	})
	if err != nil {
		log.Println(err)
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

	// handle `/` route
	fs := http.FileServer(http.Dir("./templates"))
	http.Handle("/", fs)

	log.Fatal(http.ListenAndServe(":"+port, nil))

}
