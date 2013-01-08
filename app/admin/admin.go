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
)

type Post struct {
	Sequence    int64
	Content     []byte
	ContentString string `datastore:"-"`
}

func init() {
	http.HandleFunc("/login", login)
	http.HandleFunc("/logout", logout)
	http.HandleFunc("/admin/", adminRoot)
	http.HandleFunc("/admin/post/", managePosts)
	http.HandleFunc("/admin/post/edit/", editPost)
	http.HandleFunc("/admin/post/new/", newPost)
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

func getLatestSeq(w http.ResponseWriter, r *http.Request) int64 {
	c := appengine.NewContext(r)

	var p []Post

	q := datastore.NewQuery("Post")
	Pq,_ := q.Count(c)
	q1 := datastore.NewQuery("Post").Filter("Sequence=", int64(Pq))
	q1.GetAll(c, &p)
	if p != nil {
		post := p[0]
		return post.Sequence
	}
	return 0
}

func newPost(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	var post Post
	keyP := datastore.NewIncompleteKey(c, "Post", nil)

	if csrf.ValidateToken(r, r.FormValue("CSRFToken")) {
		if r.Method == "POST" {
			c := appengine.NewContext(r)
			post.Content = []byte(r.FormValue("Content"))
			post.Sequence = getLatestSeq(w,r) + 1
			datastore.Put(c, keyP, &post)
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
