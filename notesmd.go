package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/ivolimmen/notesmd/file"
	"github.com/ivolimmen/notesmd/handlers"
)

func main() {
	dataDir := flag.String("data_dir", "notes", "Path to the directory where all the markdown files are stored.")
	port := flag.Int("port", 8080, "Port to run the webserver on.")
	style := flag.String("code_style", "monokai", "Code highlighting format to use; default is Monokai")

	flag.Parse()

	fmt.Printf("NotesMD, running on port %d, using directory %s, using code style %s\n", *port, *dataDir, *style)

	config := file.InitConfig(*dataDir, *style)

	http.HandleFunc("/upload/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlers.FileUploadHandler(w, r, config)
	}))

	http.HandleFunc("/view/", handlers.MakeHandler(handlers.ViewHandler, config))
	http.HandleFunc("/edit/", handlers.MakeHandler(handlers.EditHandler, config))
	http.HandleFunc("/save/", handlers.MakeHandler(handlers.SaveHandler, config))
	http.HandleFunc("/delete/", handlers.MakeHandler(handlers.DeleteHandler, config))
	http.HandleFunc("/special/", handlers.MakeHandler(handlers.SpecialHandler, config))

	http.Handle("/web/", http.StripPrefix("/web", http.FileServer(http.Dir("./web"))))

	dir := filepath.Join(config.DataDir, "att")
	http.Handle("/att/", http.StripPrefix("/att", http.FileServer(http.Dir(dir))))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/view/Index", http.StatusFound)
	})

	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(*port), nil))
}
