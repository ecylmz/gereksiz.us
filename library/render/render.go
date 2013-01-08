// This package is called to render the template.
package render

import (
	"appengine"
	"appengine/user"
	"bytes"
	"fmt"
	"net/http"
	"text/template" // WARNING: template/html removes the <!--[if lt IE 9]>, so we're using text/template - but this is just applied over the header and the footer so it should be safe to use.
)

type HeaderData struct {
	IsAdmin    bool
	Username   string
	IsLoggedIn bool
}

// Renders a template
func Render(w http.ResponseWriter, r *http.Request, passedTemplate *bytes.Buffer, Statuscode ...int) {
	// Add some HTTP Headers
	if len(Statuscode) == 1 {
		w.WriteHeader(Statuscode[0])
	}

	c := appengine.NewContext(r)
	u := user.Current(c)
	headerdata := HeaderData{}
	if u != nil {
		headerdata.IsLoggedIn = true
		headerdata.Username = u.String()
		if user.IsAdmin(c) {
			headerdata.IsAdmin = true
		}
	}

	// Header
	template.Must(template.ParseFiles("templates/header.html")).Execute(w, headerdata)

	// Now add the passedTemplate
	fmt.Fprintf(w, "%s", string(passedTemplate.Bytes())) // %s = the uninterpreted bytes of the string or slice

	// And now we execute the footer
	template.Must(template.ParseFiles("templates/footer.html")).Execute(w, nil)
}
