package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"syscall"
	"text/template"
	"time"

	"github.com/gorilla/mux"
)

// Addr address to serve
var Addr string

// Root is the path to directory where will be stored files needed to run
var Root string

func init() {
	flag.StringVar(&Addr, "addr", ":8080", "addres to server")
	flag.StringVar(&Root, "root", ".", "root is the path to directory where is stored manga files.")
}

func main() {
	flag.Parse()

	sigChan := make(chan os.Signal, 64)

	signal.Notify(sigChan, os.Interrupt, os.Kill, syscall.SIGHUP)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for sig := range sigChan {
			log.Printf("signal %v received\n", sig)
			cancel()
		}
	}()

	mis, cis, pis, err := buildIndex(Root)
	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}

	repo := NewMangaRepository(mis, cis, pis)

	app := NewApp(ctx, repo)

	log.Printf("Started!")

	srv := http.Server{
		Addr:    Addr,
		Handler: app.Handler(),
	}

	go func() {
		log.Fatal(srv.ListenAndServe())
	}()

	select {
	case <-ctx.Done():
		ctxTimeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := srv.Shutdown(ctxTimeout)
		if err != nil {
			log.Printf("error while shutting down the server (err: %v)", err)
		}
	}
}

type App struct {
	ctx  context.Context
	repo *MangaRepository
}

func NewApp(ctx context.Context, repo *MangaRepository) *App {
	return &App{ctx, repo}
}

func (app *App) Handler() http.Handler {
	handler := mux.NewRouter()
	handler.Path("/").Methods("GET").HandlerFunc(app.handleHome)

	handler.Path("/m/{id}").Methods("GET").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			id, err := strconv.ParseInt(vars["id"], 10, 0)
			if err != nil {
				app.notFound(w, r)
				return
			}

			app.handleManga(w, r, int(id))
		},
	)

	handler.Path("/c/{id}").Methods("GET").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			id, err := strconv.ParseInt(vars["id"], 10, 0)
			if err != nil {
				app.notFound(w, r)
				return
			}

			app.handleChapter(w, r, int(id))
		},
	)

	handler.Path("/p/{id}").Methods("GET").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			id, err := strconv.ParseInt(vars["id"], 10, 0)
			if err != nil {
				app.notFound(w, r)
				return
			}

			app.handlePage(w, r, int(id))
		},
	)

	handler.PathPrefix("/asset/").Handler(http.StripPrefix("/asset", http.FileServer(http.Dir("asset"))))
	return handler
}

func (app *App) handleManga(w http.ResponseWriter, r *http.Request, id int) {
	mi, ok := app.repo.GetManga(id)
	if !ok {
		app.notFound(w, r)
		return
	}

	chapters := make([]map[string]string, len(mi.ChapterIDs))
	for i, cid := range mi.ChapterIDs {
		ci, _ := app.repo.GetChapter(cid)
		chapters[i] = map[string]string{
			"Name": ci.Name,
			"ID":   strconv.Itoa(ci.ID),
		}
	}

	data := map[string]interface{}{
		"Name":     mi.Name,
		"ID":       mi.ID,
		"Chapters": chapters,
	}

	ExecuteTemplate(w, r, MangaTemplateName, data)
}

func (app *App) handlePage(w http.ResponseWriter, r *http.Request, id int) {
	page, ok := app.repo.GetPageByID(id)
	if !ok {
		app.notFound(w, r)
		return
	}
	fi, err := os.Stat(page.Path)
	if os.IsNotExist(err) {
		app.notFound(w, r)
		return
	}

	if err != nil {
		serverError(w, r, err)
		return
	}

	if !fi.Mode().IsRegular() {
		serverError(w, r, errors.New("not a file"))
		return
	}

	http.ServeFile(w, r, page.Path)
}

func (app *App) notFound(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

func (app *App) handleChapter(w http.ResponseWriter, r *http.Request, id int) {
	chapter, ok := app.repo.GetChapter(id)
	if !ok {
		app.notFound(w, r)
		return
	}

	manga, _ := app.repo.GetManga(chapter.ID)

	ExecuteTemplate(w, r, ChapterTemplateName, map[string]interface{}{
		"Manga":   manga,
		"Chapter": chapter,
	})
}

func (app *App) handleHome(w http.ResponseWriter, r *http.Request) {
	mis := app.repo.GetAllMangas()

	sort.Slice(mis, func(i, j int) bool {
		return mis[i].Name < mis[j].Name
	})

	ExecuteTemplate(w, r, HomeTemplateName, mis)
}

func serverError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("path: %s, error: %v", r.URL.Path, err)
	panic(http.ErrAbortHandler)
}

var tpl = template.Must(template.New("tmpl").ParseGlob("template/*.tmpl"))

// Template names
const (
	HomeTemplateName    = "home.tmpl"
	MangaTemplateName   = "manga.tmpl"
	ChapterTemplateName = "chapter.tmpl"
)

//ExecuteTemplate executes template or call serverError, if an error occur.
func ExecuteTemplate(w http.ResponseWriter, r *http.Request, name string, data interface{}) {
	err := tpl.ExecuteTemplate(w, name, data)
	if err != nil {
		serverError(w, r, err)
	}
}
