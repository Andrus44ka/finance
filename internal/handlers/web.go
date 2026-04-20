package handlers

import (
	"net/http"
	"path/filepath"
)

// WebHandler - обслуживает веб-страницу
func WebHandler(w http.ResponseWriter, r *http.Request) {
	// Отдаем HTML файл
	http.ServeFile(w, r, filepath.Join("web", "index.html"))
}

// StaticHandler - обслуживает статические файлы (CSS, JS)
func StaticHandler(w http.ResponseWriter, r *http.Request) {
	// Убираем "/static/" из пути
	path := r.URL.Path[len("/static/"):]
	http.ServeFile(w, r, filepath.Join("web", path))
}
