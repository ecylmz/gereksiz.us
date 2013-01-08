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
	"encoding/gob"
)

type Post struct {
	Sequence    int64
	Content     []byte
	ContentString string `datastore:"-"`
}

func init() {
	http.HandleFunc("/", root)
}

func randInt(min int , max int) int {
	return min + rand.Intn(max-min)
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

func root(w http.ResponseWriter, r *http.Request) {
	var postSeq int
	trimPath := strings.Trim(r.URL.Path, "/g/")
	arg, err := strconv.Atoi(trimPath)

	if trimPath != "" && err == nil {
		postSeq = arg
	} else {
		c := appengine.NewContext(r)
		qP := datastore.NewQuery("Post")
		PostCount, _ := qP.Count(c)
		postSeq = randInt(1, PostCount + 1)
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
