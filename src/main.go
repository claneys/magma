package main

import (
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/help-14/magma/modules"
)

var pwd string
var appConfig modules.Config
var websiteData = struct {
	Config   modules.WebsiteConfig
	Language modules.Language
	Contents []modules.GroupData
}{}
var webTemplate *template.Template

func main() {
	pwd, _ = os.Getwd()

	dataPath := filepath.Join(pwd, "data")
	if _, err := os.Stat(filepath.Join(dataPath, "data.yaml")); os.IsNotExist(err) {
		log.Println("Copy files to data folder")
		os.MkdirAll(dataPath, os.ModePerm)
		modules.CopyDir(filepath.Join(pwd, "common"), dataPath)
	}
	loadData()

	commonfs := http.FileServer(http.Dir(dataPath))
	http.Handle("/common/", http.StripPrefix("/common/", commonfs))

	languagefs := http.FileServer(http.Dir(filepath.Join(pwd, "languages")))
	http.Handle("/languages/", http.StripPrefix("/languages/", languagefs))

	themefs := http.FileServer(http.Dir(filepath.Join(pwd, "themes", appConfig.Website.Theme)))
	http.Handle("/theme/", http.StripPrefix("/theme/", themefs))

	http.HandleFunc("/weather", serveWeather)
	http.HandleFunc("/", serveTemplate)

	log.Println("Listening on http://localhost:7001 ...")
	err := http.ListenAndServe(":7001", nil)
	if err != nil {
		log.Fatal(err)
	}
}

func loadData() {
	appConfig = modules.LoadConfig()
	websiteData.Config = appConfig.Website
	websiteData.Language = modules.LoadLanguage(appConfig.Website.Language)
	websiteData.Contents = modules.LoadContent().Data

	tmpl, _ := template.ParseFiles(filepath.Join(pwd, "themes", appConfig.Website.Theme, "index.html"))
	webTemplate = tmpl
}

func serveTemplate(w http.ResponseWriter, r *http.Request) {
	webTemplate.Execute(w, websiteData)
}

var weatherTimeOut int64
var weatherCache []byte

func serveWeather(w http.ResponseWriter, r *http.Request) {
	if time.Now().UnixMilli() >= weatherTimeOut {
		resp, err := http.Get("https://api.openweathermap.org/data/2.5/weather?lat=" + appConfig.OpenWeatherMap.Latitude + "&lon=" + appConfig.OpenWeatherMap.Longitude + "&limit=1&appid=" + appConfig.OpenWeatherMap.ApiKey)
		if err != nil {
			log.Fatalln(err)
			return
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatalln(err)
			return
		}
		weatherCache = body
		weatherTimeOut = time.Now().UnixMilli() + 1800000
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(weatherCache)
}
