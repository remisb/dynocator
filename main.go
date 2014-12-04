package main

import (
	"encoding/json"
	"flag"
	"github.com/BurntSushi/toml"
	"github.com/gorilla/mux"
	"gopkg.in/fsnotify.v1"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var flags = ReadFlags()
var config = ReadConfig()

func init() {
	log.Print(config.Baseurl)
	ConvertAllPosts()
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/"+config.Admin, AdminIndex).Methods("GET")
	r.HandleFunc("/"+config.Admin+"/add", AddGet).Methods("GET")
	r.HandleFunc("/"+config.Admin+"/add", AddPost).Methods("POST")
	r.HandleFunc("/"+config.Admin+"/edit/{post}", EditPost).Methods("GET")
	r.HandleFunc("/"+config.Admin+"/uploadimage", UploadImage).Methods("POST")
	r.PathPrefix("/").Handler(http.FileServer(http.Dir(config.Public)))

	log.Printf("Server running on %s", ("http://localhost" + flags.Port))
	log.Printf("Press ctrl+c to stop")

	// Start file-change watcher for posts/
	go ConvertWatcher()

	http.Handle("/", r)
	http.ListenAndServe(flags.Port, nil)
}

func AdminIndex(w http.ResponseWriter, r *http.Request) {
	/*
		files, _ := filepath.Glob(config.Posts + "/*.html")

		for _, v := range files {

		}
	*/
	tmpl := template.Must(template.New("index").Funcs(funcMap).ParseGlob(config.Admin + "/*.html"))
	tmpl.Execute(w, nil)
}

func AddGet(w http.ResponseWriter, r *http.Request) {

	tmpl := template.Must(template.New("add").Funcs(funcMap).ParseGlob(config.Admin + "/*.html"))
	tmpl.Execute(w, map[string]interface{}{"Admin": config.Admin})

}

func EditPost(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	post := vars["post"]
	y := config.Posts + "/" + post + ".html"
	data, _ := ioutil.ReadFile(y)
	x := string(data)

	metadata := ReadMetaData(y)

	tmpl := template.Must(template.New("edit").Funcs(funcMap).ParseGlob(config.Admin + "/*.html"))
	tmpl.Execute(w, map[string]interface{}{"Post": x, "Admin": config.Admin, "Metadata": &metadata})

}

func AddPost(w http.ResponseWriter, r *http.Request) {

	// Slugify the title
	Title := r.FormValue("title")
	Title = string(Title)
	t := strings.Split(Title, " ")
	titleslug := strings.ToLower(strings.Join(t, "-"))

	// Author and Post
	Author := r.FormValue("author")
	Post := r.FormValue("post")

	// Save post as static html file
	filename := config.Posts + "/" + titleslug + ".html"
	f, _ := os.Create(filename)
	f.WriteString(Post)

	// Save metadata to toml file
	filename2 := config.Metadata + "/" + titleslug + ".toml"
	f2, _ := os.Create(filename2)
	f2.WriteString("title = " + "\"" + Title + "\"\n")
	f2.WriteString("author = " + "\"" + Author + "\"\n")
	f2.WriteString("date = " + "\"" + time.Now().Format("January 2, 2006 3:04PM") + "\"\n")
	f2.WriteString("slug = " + "\"" + titleslug + "\"\n")

	// Redirect to admin page
	http.Redirect(w, r, ("/admin/edit/" + titleslug), 301)

}

func UploadImage(w http.ResponseWriter, r *http.Request) {
	file, handler, err := r.FormFile("post")
	if err != nil {
		log.Println(err)
		return
	}
	defer file.Close()
	filename := config.Public + "/static/postimages/" + handler.Filename
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Println(err)
		return
	}
	defer f.Close()
	io.Copy(f, file)

	//log.Print(handler.Filename)
	filename2 := config.Baseurl + "/static/postimages/" + handler.Filename
	m := map[string]string{"link": filename2}
	m2, _ := json.Marshal(m)
	//log.Println(m2)

	w.Header().Set("Content-Type", "application/json")
	w.Write(m2)
}

var funcMap = template.FuncMap{
	"Admin": Admin,
}

func Admin() string {
	return config.Admin
}

// Info from config file
type Config struct {
	Baseurl   string
	Title     string
	Templates string
	Posts     string
	Public    string
	Admin     string
	Metadata  string
	Index     string
}

// Reads info from config file
func ReadConfig() Config {
	var configfile = flags.Configfile
	_, err := os.Stat(configfile)
	if err != nil {
		log.Fatal("Config file is missing: ", configfile)
	}

	var config Config
	if _, err := toml.DecodeFile(configfile, &config); err != nil {
		log.Fatal(err)
	}
	log.Print(config.Index)
	return config
}

// Flag stuff
type Flag struct {
	Port       string
	Configfile string
}

// Reads command-line flags
func ReadFlags() Flag {
	p := flag.String("port", ":1414", "port to run server on")
	c := flag.String("config", "config.toml", "config file ")
	flag.Parse()
	x := Flag{*p, *c}

	return x
}

// Converts all markdown posts to static pages
func ConvertAllPosts() {

	x := config.Public + "/*"
	// Read all the files
	f, _ := filepath.Glob(x)

	//Remove all folders/files in public/ except static/ and index.html
	for _, v := range f {
		boo := strings.Contains(v, config.Public+"/static")
		boo2 := strings.Contains(v, config.Public+"/index.html")
		if boo == false && boo2 == false {
			os.RemoveAll(v)
		}
	}

	p := config.Posts + "/" + "*html"
	files, _ := filepath.Glob(p)

	for _, v := range files {
		//log.Print(v)
		ConvertPost(v)

	}

}

// Converts a post to a static page
func ConvertPost(v string) {
	// Let's read some data
	dat, _ := ioutil.ReadFile(v)

	// Read file, split by lines and grab everything except first line, join lines again
	x := string(dat)

	fi := strings.TrimSuffix(v, ".html")
	fi = strings.TrimPrefix(fi, config.Posts+"/")
	fi = config.Public + "/" + fi
	os.Mkdir(fi, 0777)
	fi = fi + "/index.html"
	//log.Print(v)
	metadata := ReadMetaData(v)
	log.Print(metadata)
	r, err := os.OpenFile(fi, os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		log.Print(err)
	}
	defer r.Close()

	tmpl := template.Must(template.New("header").Funcs(funcMap).ParseGlob(config.Templates + "/*.html"))
	tmpl.Execute(r, map[string]interface{}{"Title": metadata.Title})

	tmpl2 := template.Must(template.New("single").Funcs(funcMap).ParseGlob(config.Templates + "/*.html"))
	tmpl2.Execute(r, map[string]interface{}{"Meta": &metadata, "Content": template.HTML(x), "Title": metadata.Title})

}

type Metadata struct {
	Title  string
	Author string
	Date   string
	Slug   string
}

// Reads info from config file
func ReadMetaData(v string) Metadata {
	fi := strings.TrimSuffix(v, ".html")
	fi = strings.TrimPrefix(fi, "posts/")
	fi = "metadata/" + fi + ".toml"
	var metadata Metadata
	if _, err := toml.DecodeFile(fi, &metadata); err != nil {
		log.Fatal(err)
	}

	return metadata
}

// Watches posts/ directory for changes so that static pages are built
func ConvertWatcher() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				log.Println("event:", event)
				switch {
				case event.Op == fsnotify.Create:
					log.Println("created file:", event.Name)
					ConvertAllPosts()
				case event.Op == fsnotify.Write:
					log.Println("wrote file:", event.Name)
					ConvertAllPosts()
				case event.Op == fsnotify.Chmod:
					log.Println("chmod file:", event.Name)
					ConvertAllPosts()
				case event.Op == fsnotify.Rename:
					log.Println("renamed file:", event.Name)
					ConvertAllPosts()
				case event.Op == fsnotify.Remove:
					log.Println("removed file:", event.Name)
					ConvertAllPosts()
				}

			case err := <-watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(config.Posts + "/")
	if err != nil {
		log.Fatal(err)
	}
	<-done
}
