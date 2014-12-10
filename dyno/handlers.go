package dyno

import (
	"encoding/json"
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
