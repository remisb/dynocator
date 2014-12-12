package dyno

import (
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

func Serve() {
	r := mux.NewRouter()
	r.HandleFunc("/"+config.Admin, AdminIndex).Methods("GET")
	r.HandleFunc("/"+config.Admin+"/add", AddGet).Methods("GET")
	r.HandleFunc("/"+config.Admin+"/add", AddPost).Methods("POST")
	r.HandleFunc("/"+config.Admin+"/settings", SettingsGet).Methods("GET")
	r.HandleFunc("/"+config.Admin+"/settings", SettingsPost).Methods("POST")
	r.HandleFunc("/"+config.Admin+"/edit/{post}", EditPost).Methods("GET")
	r.HandleFunc("/"+config.Admin+"/edit/{post}", UpdatePost).Methods("POST")
	r.HandleFunc("/"+config.Admin+"/uploadimage", UploadImage).Methods("POST")
	r.HandleFunc("/"+config.Admin+"/login", loginGet).Methods("GET")
	r.HandleFunc("/"+config.Admin+"/login", loginPost).Methods("POST")
	r.HandleFunc("/"+config.Admin+"/logout", logout).Methods("GET")

	r.HandleFunc("/category/{category}", Categories).Methods("GET")
	r.HandleFunc("/categories", ListCategories).Methods("GET")

	r.PathPrefix("/").Handler(http.FileServer(http.Dir(config.Public)))

	log.Printf("Server running on %s", ("http://localhost" + flags.Port))
	log.Printf("Press ctrl+c to stop")

	// Start file-change watcher for posts/
	go ConvertWatcher()

	http.Handle("/", r)
	http.ListenAndServe(flags.Port, nil)
}
