package main

import "net/http"

func makeHandler(fn func(http.ResponseWriter, *http.Request, string, Config), config Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, m[2], config)
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string, config Config) {
	p, err := loadPage(title, config)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}
	renderTemplate(w, "view", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string, config Config) {
	body := r.FormValue("body")
	p := &Page{Title: title, Raw: []byte(body)}
	err := p.save(config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

func deleteHandler(w http.ResponseWriter, r *http.Request, title string, config Config) {
	ok, err := deletePage(title, config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	if ok {
		http.Redirect(w, r, "/view/Index", http.StatusFound)
	}
}

func editHandler(w http.ResponseWriter, r *http.Request, title string, config Config) {
	p, err := loadPage(title, config)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p)
}
