package dyno

import (
	"encoding/json"
	"github.com/BurntSushi/toml"
	"github.com/gorilla/mux"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var funcMap = template.FuncMap{
	"Admin":    Admin,
	"Friendly": Friendly,
	"Ago":      Ago,
	"Baseurl":  Baseurl,
}

func Admin() string {
	return config.Admin
}

func Baseurl() string {
	return config.Baseurl
}

func Friendly(t time.Time) string {
	x := t.Format("January _2 2006 15:04:05")
	return x
}

func Ago(t time.Time) time.Duration {
	x := t.Sub(time.Now())
	return x
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

func loginGet(w http.ResponseWriter, r *http.Request) {

	params := map[interface{}]interface{}{"Admin": config.Admin, "Title": config.Title}

	CreateTemplate("login", (config.Admin + "/*.html"), w, params)

}

func loginPost(w http.ResponseWriter, r *http.Request) {

	var config = ReadConfig()

	name := r.FormValue("name")
	pass := r.FormValue("password")

	redirectTarget := "/admin/login"
	if name != "" && pass != "" {
		if name == config.Username && pass == config.Password {
			SetSession(name, w, r)
			redirectTarget = config.Baseurl + "/" + config.Admin
		}
	}
	http.Redirect(w, r, redirectTarget, 302)

}

func logout(w http.ResponseWriter, r *http.Request) {
	ClearSession(w, r)
	http.Redirect(w, r, "/admin/login", 302)
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

	var configfile = flags.Configfile
	var config = ReadConfig()

	filename := configfile
	f, _ := os.Create(filename)

	x := Config{
		Baseurl:   r.FormValue("baseurl"),
		Title:     r.FormValue("title"),
		Templates: r.FormValue("templates"),
		Posts:     r.FormValue("posts"),
		Public:    r.FormValue("public"),
		Admin:     r.FormValue("admin"),
		Metadata:  r.FormValue("metadata"),
		Index:     r.FormValue("index"),
		Username:  config.Username,
		Password:  config.Password,
	}

	if err := toml.NewEncoder(f).Encode(x); err != nil {
		log.Fatal(err)
	}

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
	Post := r.FormValue("post")
	Categories := r.FormValue("categories")
	Publish := r.FormValue("publish")
	Section := r.FormValue("section")
	//log.Print(Section)

	if Section == "" {
		Section = "/"
	}

	cat := strings.Split(strings.TrimSpace(Categories), ",")
	var cat2 []string
	for _, v := range cat {
		if v != "" {
			cat2 = append(cat2, strings.TrimSpace(v))
		}
	}

	var pub bool
	if Publish == "publish" {
		pub = true
	} else {
		pub = false
	}

	fulldir := config.Metadata + Section
	//log.Print(fulldir)

	if Section != "/" {
		os.Mkdir(fulldir, 0777)
	}

	file := config.Metadata + Section + titleslug + ".toml"
	o, _ := os.Create(file)

	x := Metadata{
		Title:      r.FormValue("title"),
		Author:     r.FormValue("author"),
		Date:       time.Now(),
		Slug:       titleslug,
		Categories: cat2,
		Publish:    pub,
		Section:    Section,
	}

	if err := toml.NewEncoder(o).Encode(x); err != nil {
		log.Fatal(err)
	}

	// Save post as static html file
	fulldir2 := config.Posts + Section
	//log.Print(fulldir2)

	if Section != "/" {
		os.Mkdir(fulldir2, 0777)
	}

	filename := config.Posts + Section + titleslug + ".html"
	f, _ := os.Create(filename)
	f.WriteString(Post)

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

		// Author and Post
		Post := r.FormValue("post")
		Categories := r.FormValue("categories")

		Publish := r.FormValue("publish")

		cat := strings.Split(strings.TrimSpace(Categories), ",")
		var cat2 []string
		for _, v := range cat {
			if v != "" {
				cat2 = append(cat2, strings.TrimSpace(v))
			}
		}

		var pub bool
		if Publish == "publish" {
			pub = true
		} else {
			pub = false
		}

		file := config.Metadata + "/" + titleslug + ".toml"
		o, _ := os.Create(file)

		x := Metadata{
			Title:      r.FormValue("title"),
			Author:     r.FormValue("author"),
			Date:       z.Date,
			Slug:       titleslug,
			Categories: cat2,
			Publish:    pub,
			Section:    r.FormValue("section"),
		}

		if err := toml.NewEncoder(o).Encode(x); err != nil {
			log.Fatal(err)
		}

		// Save post as static html file
		filename := config.Posts + "/" + titleslug + ".html"
		f, _ := os.Create(filename)
		f.WriteString(Post)

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
