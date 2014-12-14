package dyno

import (
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

func Serve() {
	r := mux.NewRouter()

	r.HandleFunc("/"+config.Admin, Auth(AdminIndex)).Methods("GET")
	r.HandleFunc("/"+config.Admin+"/add", Auth(AddGet)).Methods("GET")
	r.HandleFunc("/"+config.Admin+"/add", Auth(AddPost)).Methods("POST")
	r.HandleFunc("/"+config.Admin+"/settings", Auth(SettingsGet)).Methods("GET")
	r.HandleFunc("/"+config.Admin+"/settings", Auth(SettingsPost)).Methods("POST")
	r.HandleFunc("/"+config.Admin+"/edit/{post}", Auth(EditPost)).Methods("GET")
	r.HandleFunc("/"+config.Admin+"/edit/{post}", Auth(UpdatePost)).Methods("POST")
	r.HandleFunc("/"+config.Admin+"/uploadimage", UploadImage).Methods("POST")
	r.HandleFunc("/"+config.Admin+"/login", loginGet).Methods("GET")
	r.HandleFunc("/"+config.Admin+"/login", loginPost).Methods("POST")
	r.HandleFunc("/"+config.Admin+"/logout", logout).Methods("GET")

	r.HandleFunc("/category/{category}", Categories).Methods("GET")
	r.HandleFunc("/categories", ListCategories).Methods("GET")

	r.PathPrefix("/").Handler(http.FileServer(http.Dir(config.Public)))

	log.Printf("Server running on port %s", flags.Port)
	log.Printf("Press ctrl+c to stop")

	// Start file-change watcher for posts/
	go ConvertWatcher()

	http.Handle("/", r)
	http.ListenAndServe(flags.Port, nil)
}
