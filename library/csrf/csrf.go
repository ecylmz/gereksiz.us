package csrf

import (
	"appengine"
	"appengine/datastore"
	"appengine/memcache"
	"appengine/user"
	"crypto/rand"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
)

/* For the CSRF Token */
type SecurityToken struct {
	Token string
}

/*
 * Function to return a random string
 */
func makeRandomString(size int) string {
	var buf []byte = make([]byte, size)

	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		return ""
	}

	var encbuf []byte = make([]byte, base64.StdEncoding.EncodedLen(len(buf)))
	base64.StdEncoding.Encode(encbuf, buf)

	return strings.Replace(string(encbuf), "+", "_", -1)
}

/*
 * This exported functions returns (and if needed, generates) a token
 */
func GetToken(r *http.Request) string {
	c := appengine.NewContext(r)
	u := user.Current(c)

	// CSRF Token name
	csrfTokenUserID := "CSRF" + u.ID

	// Get the item from the memcache
	if item, err := memcache.Get(c, csrfTokenUserID); err == nil {
		// Return the token
		return string(item.Value)
	}
	// Get the item from the datastore
	key := datastore.NewKey(c, "SecurityToken", csrfTokenUserID, 0, nil)
	var datastoreSecurityToken SecurityToken
	if err := datastore.Get(c, key, &datastoreSecurityToken); err == nil {
		// Add it to memcache
		item := &memcache.Item{
			Key:   csrfTokenUserID,
			Value: []byte(datastoreSecurityToken.Token),
		}
		memcache.Add(c, item)
		// Return the token
		return datastoreSecurityToken.Token
	}

	// Generate a item
	csrfToken := makeRandomString(256)
	// Save it to the memcache
	item := &memcache.Item{
		Key:   csrfTokenUserID,
		Value: []byte(csrfToken),
	}
	memcache.Add(c, item)
	// Save it to the datastore
	SecurityToken := SecurityToken{
		Token: csrfToken,
	}
	datastore.Put(c, datastore.NewKey(c, "SecurityToken", u.ID, 0, nil), &SecurityToken)

	return csrfToken
}

/*
 * Validate CSRF Token
 */
func ValidateToken(r *http.Request, token string) bool {
	c := appengine.NewContext(r)
	u := user.Current(c)
	if token == "" {
		return false
	}

	// CSRF Token name
	csrfTokenUserID := "CSRF" + u.ID
	// Check if we have a token in the memcache 
	// Yes there is a token, make it readable
	// Get the item from the memcache
	item, _ := memcache.Get(c, csrfTokenUserID)

	// Let's make a lookup
	if token == string(item.Value) {
		// They match!
		return true
	}
	// No token, let's have a look in the datastore
	key := datastore.NewKey(c, "SecurityToken", csrfTokenUserID, 0, nil)
	var datastoreSecurityToken SecurityToken
	if err := datastore.Get(c, key, &datastoreSecurityToken); err != nil {
		// Error, return false
		return false
	}
	// Let's make a lookup
	if token == datastoreSecurityToken.Token {
		// They match!
		return true
	}
	return false
}
