package recaptcha

import (
	"appengine"
	"appengine/urlfetch"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

// Returns true if the captcha is right
func Validate(r *http.Request, remoteAddr string, challenge string, response string) bool {
	privatekey := "6Ldmi9sSAAAAAF2XrEh6xq0Fq19YQgy0aXKl4G_R"
	c := appengine.NewContext(r)
	client := urlfetch.Client(c)

	resp, err := client.PostForm("http://www.google.com/recaptcha/api/verify",
		url.Values{"privatekey": {privatekey}, "remoteip": {remoteAddr}, "challenge": {challenge}, "response": {response}})
	if err != nil {
		return false
	}

	defer resp.Body.Close()
	item, _ := ioutil.ReadAll(resp.Body)
	items := string(item)

	if strings.Contains(items, "true") == true {
		return true
	}
	return false
}
