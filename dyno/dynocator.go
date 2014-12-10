// Dynocator is a lightweight blog/website engine written in Go
package dyno

import (
	"flag"
	"github.com/BurntSushi/toml"
	"gopkg.in/fsnotify.v1"
	"html/template"
	"io/ioutil"
	"log"
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

// Refresh converts all posts to static pages and generates and index page
func Refresh() {
	ConvertAllPosts()
	Index()
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

	var config = ReadConfig()

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
