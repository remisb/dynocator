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
	"sort"
	"strings"
	"time"
)

var flags = ReadFlags()
var config = ReadConfig()

func init() {

	t := time.Now()

	Refresh()

	t2 := time.Now()
	diff := t2.Sub(t)

	log.Printf("In %s", diff)
}

func Refresh() {
	ConvertAllPosts()
	Index()
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/"+config.Admin, AdminIndex).Methods("GET")
	r.HandleFunc("/"+config.Admin+"/add", AddGet).Methods("GET")
	r.HandleFunc("/"+config.Admin+"/add", AddPost).Methods("POST")
	r.HandleFunc("/"+config.Admin+"/settings", SettingsGet).Methods("GET")
	r.HandleFunc("/"+config.Admin+"/settings", SettingsPost).Methods("POST")
	r.HandleFunc("/"+config.Admin+"/edit/{post}", EditPost).Methods("GET")
	r.HandleFunc("/"+config.Admin+"/edit/{post}", UpdatePost).Methods("POST")
	r.HandleFunc("/"+config.Admin+"/uploadimage", UploadImage).Methods("POST")

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

func CreateTemplate(name string, globpath string, w io.Writer, params map[interface{}]interface{}) {
	tmpl := template.Must(template.New(name).Funcs(funcMap).ParseGlob(globpath))
	tmpl.Execute(w, params)
}

func AdminIndex(w http.ResponseWriter, r *http.Request) {

	posts := ExtractPostsByDate()

	var meta []Metadata

	for _, v := range posts {
		info := ReadMetaData(v)
		meta = append(meta, info)

	}
	//log.Print(meta)
	params := map[interface{}]interface{}{"Posts": &meta, "Title": "Admin"}

	CreateTemplate("index", (config.Admin + "/*.html"), w, params)
}

func AddGet(w http.ResponseWriter, r *http.Request) {

	params := map[interface{}]interface{}{"Admin": config.Admin, "Title": config.Title}

	CreateTemplate("add", (config.Admin + "/*.html"), w, params)

}

func SettingsGet(w http.ResponseWriter, r *http.Request) {

	var conf = ReadConfig()

	params := map[interface{}]interface{}{"Config": conf, "Title": config.Title}

	CreateTemplate("settings", (config.Admin + "/*.html"), w, params)

}

func SettingsPost(w http.ResponseWriter, r *http.Request) {

	baseurl := r.FormValue("baseurl")
	title := r.FormValue("title")
	templates := r.FormValue("templates")
	posts := r.FormValue("posts")
	public := r.FormValue("public")
	admin := r.FormValue("admin")
	metadata := r.FormValue("metadata")
	index := r.FormValue("index")

	var configfile = flags.Configfile
	filename := configfile
	f2, _ := os.Create(filename)

	f2.WriteString("baseurl = " + "\"" + baseurl + "\"\n")
	f2.WriteString("title = " + "\"" + title + "\"\n")
	f2.WriteString("templates = " + "\"" + templates + "\"\n")
	f2.WriteString("posts = " + "\"" + posts + "\"\n")
	f2.WriteString("public = " + "\"" + public + "\"\n")
	f2.WriteString("admin = " + "\"" + admin + "\"\n")
	f2.WriteString("metadata = " + "\"" + metadata + "\"\n")
	f2.WriteString("index = " + "\"" + index + "\"")

	ConvertAllPosts()
	Index()

	http.Redirect(w, r, ("/admin/settings"), 301)

}

func EditPost(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	post := vars["post"]
	y := config.Posts + "/" + post + ".html"
	data, _ := ioutil.ReadFile(y)
	x := string(data)

	metadata := ReadMetaData(y)

	params := map[interface{}]interface{}{"Post": x, "Admin": config.Admin, "Metadata": &metadata, "Title": "Edit Post", "Baseurl": config.Baseurl}

	CreateTemplate("edit", (config.Admin + "/*.html"), w, params)

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
	Categories := r.FormValue("categories")
	Publish := r.FormValue("publish")

	// Save post as static html file
	filename := config.Posts + "/" + titleslug + ".html"
	f, _ := os.Create(filename)
	f.WriteString(Post)

	// Save metadata to toml file
	filename2 := config.Metadata + "/" + titleslug + ".toml"
	f2, _ := os.Create(filename2)
	f2.WriteString("title = " + "\"" + Title + "\"\n")
	f2.WriteString("author = " + "\"" + Author + "\"\n")

	// Change time format to Zulu - hack, there's gotta be a better way to do this
	z := string(time.Now().Format(time.RFC3339 + "Z"))
	k := strings.Split(z, "")
	y := k[:19]
	o := strings.Join(y, "")
	o = o + "Z"
	//z = strings.Replace(z, "+", "Z", 1)
	f2.WriteString("date = " + " " + o + "\n")
	f2.WriteString("slug = " + "\"" + titleslug + "\"\n")
	f2.WriteString("categories = " + "[")

	cat := strings.Split(strings.TrimSpace(Categories), ",")
	for _, v := range cat {
		f2.WriteString("\"" + v + "\"" + ",")
	}
	f2.WriteString("]\n")

	if Publish == "publish" {
		f2.WriteString("publish = " + "true" + "\n")
	} else {
		f2.WriteString("publish = " + "false" + "\n")
	}

	// Redirect to admin page
	http.Redirect(w, r, ("/admin"), 301)

}

func UpdatePost(w http.ResponseWriter, r *http.Request) {

	Submit := r.FormValue("submit")

	if Submit == "save" {

		// Slugify the title
		Title := r.FormValue("title")
		Title = string(Title)
		t := strings.Split(Title, " ")
		titleslug := strings.ToLower(strings.Join(t, "-"))

		z := ReadMetaData(titleslug)
		zz := z.Date.Format(time.RFC3339)
		//log.Print(zz)

		// Author and Post
		Author := r.FormValue("author")
		Post := r.FormValue("post")
		Categories := r.FormValue("categories")

		Publish := r.FormValue("publish")

		// Save post as static html file
		filename := config.Posts + "/" + titleslug + ".html"
		f, _ := os.Create(filename)
		f.WriteString(Post)

		// Save metadata to toml file
		filename2 := config.Metadata + "/" + titleslug + ".toml"
		f2, _ := os.Create(filename2)
		f2.WriteString("title = " + "\"" + Title + "\"\n")
		f2.WriteString("author = " + "\"" + Author + "\"\n")

		// Change time format to Zulu - hack, there's gotta be a better way to do this

		//z = strings.Replace(z, "+", "Z", 1)
		//log.Print(z.Date)
		f2.WriteString("date = " + " " + zz + "\n")
		f2.WriteString("slug = " + "\"" + titleslug + "\"\n")

		f2.WriteString("categories = " + "[")

		cat := strings.Split(strings.TrimSpace(Categories), ",")
		var cat2 []string
		for k, v := range cat {
			if v != "" {
				cat2 = append(cat2, strings.TrimSpace(v))
				log.Print(k, v)
			}
		}

		for _, v := range cat2 {
			f2.WriteString("\"" + v + "\"" + ",")
		}
		f2.WriteString("]\n")

		if Publish == "publish" {
			f2.WriteString("publish = " + "true" + "\n")
		} else {
			f2.WriteString("publish = " + "false" + "\n")
		}

		// Redirect to admin page
		http.Redirect(w, r, ("/admin"), 301)

	} else if Submit == "delete" {

		// Slugify the title
		Title := r.FormValue("title")
		Title = string(Title)
		t := strings.Split(Title, " ")
		titleslug := strings.ToLower(strings.Join(t, "-"))
		log.Print(titleslug)

		os.Remove(config.Posts + "/" + titleslug + ".html")
		os.Remove(config.Metadata + "/" + titleslug + ".toml")

		// Redirect to admin page
		http.Redirect(w, r, ("/admin"), 301)
	}

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

func Categories(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	category := vars["category"]

	posts := ExtractPostsByDate()

	var published []string

	for _, v := range posts {
		info := ReadMetaData(v)
		if info.Publish == true {
			published = append(published, v)
		}
	}
	//log.Print(posts)
	var cats []string
	for _, v := range published {
		met := ReadMetaData(v)
		//log.Print(met.Categories)
		for _, n := range met.Categories {
			if strings.TrimSpace(n) == category {
				cats = append(cats, v)
			}
		}
	}

	var meta []Post

	for _, v := range cats {

		filename := config.Posts + "/" + v + ".html"

		data, _ := ioutil.ReadFile(filename)
		x := string(data)
		y := strings.Split(x, "<p class=\"fr-tag\">")

		var summ string
		if len(y) == 1 {
			summ = ""
		} else {
			yy := y[:2]
			summ = strings.Join(yy, " ")
		}

		info := ReadMetaData(v)
		meta = append(meta, Post{
			Title:   info.Title,
			Author:  info.Author,
			Date:    info.Date,
			Slug:    info.Slug,
			Summary: template.HTML(summ),
		})

	}

	params := map[interface{}]interface{}{"Posts": &meta, "Title": config.Title}

	CreateTemplate("category", (config.Templates + "/*.html"), w, params)

}

func ListCategories(w http.ResponseWriter, r *http.Request) {

	posts := ExtractPostsByDate()

	var published []string

	for _, v := range posts {
		info := ReadMetaData(v)
		if info.Publish == true {
			published = append(published, v)
		}
	}

	var cats []string
	for _, v := range published {
		info := ReadMetaData(v)
		for _, x := range info.Categories {
			cats = append(cats, strings.TrimSpace(x))
		}
	}

	fats := UniqStr(cats)

	params := map[interface{}]interface{}{"Categories": fats, "Title": config.Title}

	CreateTemplate("categories", (config.Templates + "/*.html"), w, params)

}

func UniqStr(col []string) []string {
	m := map[string]struct{}{}
	for _, v := range col {
		if _, ok := m[v]; !ok {
			m[v] = struct{}{}
		}
	}
	list := make([]string, len(m))

	i := 0
	for v := range m {
		list[i] = v
		i++
	}
	return list
}

type Post struct {
	Title   string
	Author  string
	Date    time.Time
	Slug    string
	Summary template.HTML
}

func Index() {
	var conf = ReadConfig()
	if conf.Index == "default" {
		CreateIndex()
	} else {
		CreateSlugIndex(config.Index)
	}
}

func CreateIndex() {

	posts := ExtractPostsByDate()

	var published []string

	for _, v := range posts {
		info := ReadMetaData(v)
		if info.Publish == true {
			published = append(published, v)
		}
	}

	var meta []Post

	for _, v := range published {

		filename := config.Posts + "/" + v + ".html"

		data, _ := ioutil.ReadFile(filename)
		x := string(data)
		y := strings.Split(x, "<p class=\"fr-tag\">")

		var summ string
		if len(y) == 1 {
			summ = ""
		} else {
			yy := y[:2]
			summ = strings.Join(yy, " ")
		}

		info := ReadMetaData(v)
		meta = append(meta, Post{
			Title:   info.Title,
			Author:  info.Author,
			Date:    info.Date,
			Slug:    info.Slug,
			Summary: template.HTML(summ),
		})

	}
	//log.Print(meta)

	os.Remove(config.Public + "/index.html")
	r, err := os.OpenFile(config.Public+"/index.html", os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		log.Print(err)
	}
	defer r.Close()

	params := map[interface{}]interface{}{"Posts": &meta, "Title": config.Title}

	CreateTemplate("index", (config.Templates + "/*.html"), r, params)

	log.Printf("Index page generated")
}

func CreateSlugIndex(v string) {

	filename := config.Posts + "/" + v + ".html"
	dat, _ := ioutil.ReadFile(filename)

	// Read file, split by lines and grab everything except first line, join lines again
	x := string(dat)

	os.Remove(config.Public + "/index.html")
	r, err := os.OpenFile(config.Public+"/index.html", os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		log.Print(err)
	}
	defer r.Close()

	metadata := ReadMetaData(v)

	params := map[interface{}]interface{}{"Metadata": &metadata, "Content": template.HTML(x), "Title": metadata.Title}

	CreateTemplate("single", (config.Templates + "/*.html"), r, params)

}

var funcMap = template.FuncMap{
	"Admin":    Admin,
	"Friendly": Friendly,
	"Ago":      Ago,
}

func Admin() string {
	return config.Admin
}

func Friendly(t time.Time) string {
	x := t.Format("January _2 2006 15:04:05")
	return x
}

func Ago(t time.Time) time.Duration {
	x := t.Sub(time.Now())
	return x
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
	//log.Print(config.Index)
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
	i := 0

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
		i++
	}

	log.Printf("%d pages created", i)
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
	//log.Print(metadata)
	r, err := os.OpenFile(fi, os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		log.Print(err)
	}
	defer r.Close()

	params := map[interface{}]interface{}{"Metadata": &metadata, "Content": template.HTML(x), "Title": metadata.Title}

	CreateTemplate("single", (config.Templates + "/*.html"), r, params)

}

type Metadata struct {
	Title      string
	Author     string
	Date       time.Time
	Slug       string
	Categories []string
	Publish    bool
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
					Refresh()
				case event.Op == fsnotify.Write:
					log.Println("wrote file:", event.Name)
					Refresh()
				case event.Op == fsnotify.Chmod:
					log.Println("chmod file:", event.Name)
					Refresh()
				case event.Op == fsnotify.Rename:
					log.Println("renamed file:", event.Name)
					Refresh()
				case event.Op == fsnotify.Remove:
					log.Println("removed file:", event.Name)
					Refresh()
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
	err = watcher.Add(config.Metadata + "/")
	if err != nil {
		log.Fatal(err)
	}
	<-done
}

func ExtractPostsByDate() []string {

	files, _ := filepath.Glob(config.Posts + "/*.html")

	dates := map[time.Time]string{}

	for _, v := range files {

		data := ReadMetaData(v)
		dates[data.Date] = data.Slug

	}

	// Store keys from map
	var keys []time.Time
	for k := range dates {
		keys = append(keys, k)
	}
	sort.Sort(TimeSlice(keys))

	// Finally copy to new slice with sorted values
	tm2 := []string{}
	i := 0
	for _, k := range keys {
		tm2 = append(tm2, dates[k])
		i++
	}

	return tm2

}

//
type TimeSlice []time.Time

// Forward request for length
func (p TimeSlice) Len() int {
	return len(p)
}

// Define compare
func (p TimeSlice) Less(i, j int) bool {
	return p[i].After(p[j])
}

// Define swap over an array
func (p TimeSlice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}
