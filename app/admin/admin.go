// Copyright 2013 Emre Can Yılmaz <ecylmz@ecylmz.com>.

package admin

import (
	"appengine"
	"appengine/user"
	"appengine/datastore"
	"net/http"
	"text/template"
	"library/render"
	"bytes"
	"library/csrf"
	"strings"
	"strconv"
	"fmt"
	"library/cache"
	"encoding/gob"
	"time"
)

type Post struct {
	Sequence       int64
	Content        []byte
	ContentString  string `datastore:"-"`
}

type Counter struct {
	Count int64
}

type PostSuggestion struct {
	ID             int64 `datastore:"-"`
	Username       string
	Content        []byte
	ContentString  string `datastore:"-"`
	Timestamps     time.Time
}

func init() {
	http.HandleFunc("/login", login)
	http.HandleFunc("/logout", logout)
	http.HandleFunc("/admin/", adminRoot)
	http.HandleFunc("/admin/post/", managePosts)
	http.HandleFunc("/admin/post/edit/", editPost)
	http.HandleFunc("/admin/post/new/", newPost)
	http.HandleFunc("/admin/post/suggestion", manageSuggestions)
	http.HandleFunc("/admin/post/suggestion/edit/", editSuggestion)
	http.HandleFunc("/admin/post/suggestion/delete/", deleteSuggestion)
	http.HandleFunc("/admin/post/suggestion/accept/", acceptSuggestion)
	// http.HandleFunc("/admin/countLoad", countLoad)
	// http.HandleFunc("/admin/countLoad", countLoad)
	// http.HandleFunc("/admin/countLoad", countLoad)
}

func login(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	current_user := user.Current(c)
	if current_user == nil { login_url, _ := user.LoginURL(c, "/admin/")
		http.Redirect(w, r, login_url, http.StatusFound)
		return
	} else {
		http.Redirect(w, r, "/admin/", http.StatusFound)
		return
	}
}

func logout(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	current_user := user.Current(c)
	if current_user != nil { logout_url, _ := user.LogoutURL(c, "/")
		http.Redirect(w, r, logout_url, http.StatusFound)
		return
	} else {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
}

func adminRoot(w http.ResponseWriter, r *http.Request) {
	passedTemplate := new(bytes.Buffer)
	template.Must(template.ParseFiles("templates/admin/sidebar.html")).Execute(passedTemplate,nil)
	template.Must(template.ParseFiles("templates/admin/index.html")).Execute(passedTemplate, nil)
	render.Render(w, r, passedTemplate)
}

func getAllSuggestions(w http.ResponseWriter, r *http.Request) []PostSuggestion {
	c := appengine.NewContext(r)
	q := datastore.NewQuery("PostSuggestion").Order("-Timestamps")
	var suggestions []PostSuggestion
	keys, err := q.GetAll(c, &suggestions)
	if err != nil {
		fmt.Fprintln(w, err)
	}
	models := make([]PostSuggestion, len(suggestions))
	for i := 0; i < len(suggestions); i++ {
		models[i].ID = keys[i].IntID()
		models[i].Username = suggestions[i].Username
		models[i].ContentString = string(suggestions[i].Content)
		models[i].Timestamps = suggestions[i].Timestamps
	}

	fmt.Println(len(suggestions))
	return models
}

func manageSuggestions(w http.ResponseWriter, r *http.Request){
	type PassedData struct {
		CSRFToken string
		Suggestions []PostSuggestion
	}

	passedData := PassedData{
		CSRFToken: csrf.GetToken(r),
		Suggestions: getAllSuggestions(w, r),
	}

	passedTemplate := new(bytes.Buffer)
	template.Must(template.ParseFiles("templates/admin/sidebar.html")).Execute(passedTemplate,nil)
	template.Must(template.ParseFiles("templates/admin/suggestion/index.html")).Execute(passedTemplate, passedData)
	render.Render(w, r, passedTemplate)
}

func editSuggestion(w http.ResponseWriter, r *http.Request) {
	trimPath := strings.Trim(r.URL.Path, "/admin/post/suggestion/edit/")

	postID, _ := strconv.Atoi(trimPath)
	postID64 := int64(postID)

	c := appengine.NewContext(r)
	key := datastore.NewKey(c, "PostSuggestion", "", postID64, nil)

	var suggestion PostSuggestion
	datastore.Get(c, key, &suggestion)


	if suggestion.Content != nil {
		if csrf.ValidateToken(r, r.FormValue("CSRFToken")) {
			if r.Method == "POST" {
				c := appengine.NewContext(r)
				suggestion.Content = []byte(r.FormValue("ContentString"))
				datastore.Put(c, datastore.NewKey(c, "PostSuggestion", "", postID64, nil), &suggestion)
				http.Redirect(w, r, r.Referer(), http.StatusFound)

			}
		}

		suggestion.ID = postID64
		suggestion.ContentString = string(suggestion.Content)

		type PassedData struct {
			CSRFToken string
			Suggestion  PostSuggestion
		}

		passedData := PassedData{
			CSRFToken: csrf.GetToken(r),
			Suggestion: suggestion,
		}

		passedTemplate := new(bytes.Buffer)
		template.Must(template.ParseFiles("templates/admin/sidebar.html")).Execute(passedTemplate,nil)
		template.Must(template.ParseFiles("templates/admin/suggestion/edit.html")).Execute(passedTemplate, passedData)
		render.Render(w, r, passedTemplate)
	} else {
		fmt.Fprintln(w, "Böyle bir bilgi yok bro")
	}
}

func acceptSuggestion(w http.ResponseWriter, r *http.Request) {
	if csrf.ValidateToken(r, r.FormValue("CSRFToken")) {
		trimPath := strings.Trim(r.URL.Path, "/admin/post/suggestion/accept")

		postID, _ := strconv.Atoi(trimPath)
		postID64 := int64(postID)

		c := appengine.NewContext(r)

		keyS := datastore.NewKey(c, "PostSuggestion", "", postID64, nil)

		var suggestion PostSuggestion
		datastore.Get(c, keyS, &suggestion)

		var post Post
		keyP := datastore.NewIncompleteKey(c, "Post", nil)
		var counter Counter
		keyC := datastore.NewKey(c, "Counter", "", 1, nil)
		datastore.Get(c, keyC, &counter)
		counter.Count = counter.Count + 1

		// Add Cache Counter
		mCount := new(bytes.Buffer)
		encCount := gob.NewEncoder(mCount)
		encCount.Encode(counter)
		cache.AddCache(r, "Counter", mCount.Bytes())

		post.Content = suggestion.Content
		post.Sequence = counter.Count
		datastore.Put(c, keyP, &post)
		datastore.Put(c, keyC, &counter)
		datastore.Delete(c, keyS)
		http.Redirect(w, r, "/admin/post/suggestion", http.StatusFound)
	}
}

func deleteSuggestion(w http.ResponseWriter, r *http.Request) {
	if csrf.ValidateToken(r, r.FormValue("CSRFToken")) {
		if r.Method == "POST" {
			trimPath := strings.Trim(r.URL.Path, "/admin/post/suggestion/delete/")

			postID, _ := strconv.Atoi(trimPath)
			postID64 := int64(postID)

			c := appengine.NewContext(r)
			key := datastore.NewKey(c, "PostSuggestion", "", postID64, nil)

			datastore.Delete(c, key)

			http.Redirect(w, r, "/admin/post/suggestion", http.StatusFound)
		}}
	}

func managePosts(w http.ResponseWriter, r *http.Request) {
	if csrf.ValidateToken(r, r.FormValue("CSRFToken")) {
		if r.Method == "POST" {
			http.Redirect(w, r, "/admin/post/edit/"+ r.FormValue("Sequence"), http.StatusFound)
		}
	}
	type PassedData struct {
		CSRFToken string
	}

	passedData := PassedData{
		CSRFToken: csrf.GetToken(r),
	}

	passedTemplate := new(bytes.Buffer)
	template.Must(template.ParseFiles("templates/admin/sidebar.html")).Execute(passedTemplate,nil)
	template.Must(template.ParseFiles("templates/admin/post/index.html")).Execute(passedTemplate, passedData)
	render.Render(w, r, passedTemplate)
}

func editPost(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	// post id'sini al
	trimPath := strings.Trim(r.URL.Path, "/admin/post/edit/")

	postSeq, _ := strconv.Atoi(trimPath)
	postSeq64 := int64(postSeq)

	var p []Post

	q := datastore.NewQuery("Post").Filter("Sequence=", postSeq64)
	keys,_ := q.GetAll(c, &p)
	if p != nil {

		post := p[0]

		if csrf.ValidateToken(r, r.FormValue("CSRFToken")) {
			if r.Method == "POST" {
				c := appengine.NewContext(r)
				post.Content = []byte(r.FormValue("ContentString"))
				datastore.Put(c, datastore.NewKey(c, "Post", "", keys[0].IntID(), nil), &post)
				cache.DeleteCache(r, "post-"+strconv.Itoa(postSeq))
			}
		}

		post.ContentString = string(post.Content)

		type PassedData struct {
			CSRFToken string
			Post  Post
		}

		passedData := PassedData{
			CSRFToken: csrf.GetToken(r),
			Post: post,
		}

		passedTemplate := new(bytes.Buffer)
		template.Must(template.ParseFiles("templates/admin/sidebar.html")).Execute(passedTemplate,nil)
		template.Must(template.ParseFiles("templates/admin/post/edit.html")).Execute(passedTemplate, passedData)
		render.Render(w, r, passedTemplate)
	} else {
		fmt.Fprintln(w, "Böyle bir bilgi yok bro")
	}
}

func getCount(w http.ResponseWriter, r *http.Request) int64 {
	cachedItem, cacheStatus := cache.GetCache(r, "Counter")
	if cacheStatus == true {
		var counter Counter
		buffCount := bytes.NewBuffer(cachedItem)
		decCount := gob.NewDecoder(buffCount)
		decCount.Decode(&counter)
		return counter.Count
	}

	c := appengine.NewContext(r)
	var counter Counter
	key := datastore.NewKey(c, "Counter", "", 1, nil)
	datastore.Get(c, key, &counter)

	if counter.Count == 0 {
		// AddCache
		mCount := new(bytes.Buffer)
		encCount := gob.NewEncoder(mCount)
		encCount.Encode(counter)
		cache.AddCache(r, "Counter", mCount.Bytes())

		return counter.Count
	}

	return 0
}

// func countLoad(w http.ResponseWriter, r *http.Request) {
// 	c := appengine.NewContext(r)
// 	keyC := datastore.NewKey(c, "Counter", "", 1, nil)
// 	var counter Counter
// 	counter.Count = 127
// 	datastore.Put(c, keyC, &counter)
// }

func newPost(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	u := user.Current(c)
	if u != nil {
		if csrf.ValidateToken(r, r.FormValue("CSRFToken")) {
			if r.Method == "POST" {
				var post Post
				keyP := datastore.NewIncompleteKey(c, "Post", nil)
				var counter Counter
				keyC := datastore.NewKey(c, "Counter", "", 1, nil)
				datastore.Get(c, keyC, &counter)
				counter.Count = counter.Count + 1

				// Add Cache Counter
				mCount := new(bytes.Buffer)
				encCount := gob.NewEncoder(mCount)
				encCount.Encode(counter)
				cache.AddCache(r, "Counter", mCount.Bytes())

				c := appengine.NewContext(r)
				post.Content = []byte(r.FormValue("Content"))
				post.Sequence = counter.Count
				datastore.Put(c, keyP, &post)
				datastore.Put(c, keyC, &counter)
			}
		}
	}

	type PassedData struct {
		CSRFToken string
	}

	passedData := PassedData{
		CSRFToken: csrf.GetToken(r),
	}

	passedTemplate := new(bytes.Buffer)
	template.Must(template.ParseFiles("templates/admin/sidebar.html")).Execute(passedTemplate, nil)
	template.Must(template.ParseFiles("templates/admin/post/new.html")).Execute(passedTemplate, passedData)
	render.Render(w, r, passedTemplate)
}
