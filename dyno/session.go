package dyno

import (
	"github.com/gorilla/sessions"
	"net/http"
)

var store = sessions.NewFilesystemStore("sessions", []byte("something-very-secret"))

func SetSession(userName string, w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	session.Values["username"] = userName
	// Save it.
	session.Save(r, w)
}

func GetUserName(r *http.Request) interface{} {
	session, _ := store.Get(r, "session")
	user := session.Values["username"]
	return user
}

func ClearSession(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	session.Values["username"] = nil
	session.Save(r, w)
}
