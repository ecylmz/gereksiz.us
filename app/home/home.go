// Copyright 2013 Emre Can Yılmaz <ecylmz@ecylmz.com>.

package home

import (
	"appengine"
	"appengine/datastore"
	"net/http"
	"text/template"
	"library/render"
	"bytes"
	// "errors"
	"strings"
	"math/rand"
	"strconv"
	"fmt"
	"library/cache"
	"library/recaptcha"
	"library/csrf"
	"encoding/gob"
	"time"
)

type Post struct {
	Sequence    int64
	Content     []byte
	ContentString string `datastore:"-"`
}

type Counter struct {
	Count int64
}

type PostSuggestion struct {
	Username    string
	Content     []byte
	Timestamps  time.Time
}

func init() {
	http.HandleFunc("/", root)
	http.HandleFunc("/suggestion", suggestion)
}

func randInt(min int , max int) int {
	return min + rand.Intn(max-min)
}

func suggestion(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	if r.FormValue("ContentString") != "" {
		if recaptcha.Validate(r, r.RemoteAddr, r.FormValue("recaptcha_challenge_field"), r.FormValue("recaptcha_response_field")) == true {
			if csrf.ValidateToken(r, r.FormValue("CSRFToken")) {
				var post PostSuggestion
				post.Username    = r.FormValue("Username")
				post.Content = []byte(r.FormValue("ContentString"))
				post.Timestamps = time.Now().Local()
				datastore.Put(c, datastore.NewIncompleteKey(c, "PostSuggestion", nil), &post)
			} else {
				fmt.Println(w,"Bu İşlem İçin Yetkin Yok!")
				return
			}
		} else {
			fmt.Fprintln(w, "Captcha Kodu Yanlış! Lütfen Tekrar Dene!")
			return
		}
	}


	type PassedData struct {
		CSRFToken string
	}

	passedData := PassedData{
		CSRFToken: csrf.GetToken(r),
	}

	passedTemplate := new(bytes.Buffer)
	template.Must(template.ParseFiles("templates/suggestion.html")).Execute(passedTemplate, passedData)
	render.Render(w, r, passedTemplate)
}

func getPost(w http.ResponseWriter, r *http.Request, Seq int) Post {
	cachedItem, cacheStatus := cache.GetCache(r, "post-"+strconv.Itoa(Seq))
	if cacheStatus == true {
		var post Post
		buffPost := bytes.NewBuffer(cachedItem)
		decPost  := gob.NewDecoder(buffPost)
		decPost.Decode(&post)
		return post
	}

	c := appengine.NewContext(r)
	q := datastore.NewQuery("Post").Filter("Sequence=", Seq)
	var p []Post
	q.GetAll(c, &p)

	if p != nil {
		post := p[0]
		post.ContentString = string(post.Content)
		// Add Cache
		mPost := new(bytes.Buffer)
		encPost := gob.NewEncoder(mPost)
		encPost.Encode(post)
		cache.AddCache(r, "post-"+strconv.Itoa(Seq), mPost.Bytes())

		return post
	}
	var nilPost Post
	return nilPost
}

func getCount(w http.ResponseWriter, r *http.Request) int64 {
	cachedItem, cacheStatus := cache.GetCache(r, "Counter")
	if cacheStatus == true {
		var counter Counter
		buffCount := bytes.NewBuffer(cachedItem)
		decCount  := gob.NewDecoder(buffCount)
		decCount.Decode(&counter)
		return counter.Count
	}

	c := appengine.NewContext(r)
	var counter Counter
	key := datastore.NewKey(c, "Counter", "", 1, nil)
	datastore.Get(c, key, &counter)

	if counter.Count != 0 {
		// Add Cache
		mCount := new(bytes.Buffer)
		encCount := gob.NewEncoder(mCount)
		encCount.Encode(counter)
		cache.AddCache(r, "Counter", mCount.Bytes())

		return counter.Count
	}

	return 0
}

func root(w http.ResponseWriter, r *http.Request) {
	var postSeq int
	trimPath := strings.Trim(r.URL.Path, "/g/")
	arg, err := strconv.Atoi(trimPath)

	if trimPath != "" && err == nil {
		postSeq = arg
	} else {
		counter := getCount(w, r)
		fmt.Println(counter)
		postSeq = randInt(1, int(counter)+1)
	}

	post := getPost(w, r, postSeq)

	if post.Sequence != 0 {
		type PassedData struct {
			Post Post
		}

		passedData := PassedData{
			Post: post,
		}

		passedTemplate := new(bytes.Buffer)
		template.Must(template.ParseFiles("templates/index.html")).Execute(passedTemplate, passedData)
		render.Render(w, r, passedTemplate)
	} else {
		fmt.Fprintln(w, "Öyle bir bilgi yok bro.")
	}
}
