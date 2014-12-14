package dyno

import (
	"net/http"
)

func Auth(handler http.HandlerFunc) http.HandlerFunc {
	x := func(w http.ResponseWriter, r *http.Request) {
		UserName := GetUserName(r)
		if UserName == nil {
			http.Redirect(w, r, "/admin/login", 302)
		} else {
			handler.ServeHTTP(w, r)
		}
	}
	return x
}
