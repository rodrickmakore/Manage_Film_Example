package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"

	// CONFIG
	"skaffolder/Manage_Film_Example/config"
	"skaffolder/Manage_Film_Example/security"

	// start import model

	// APIs
	"skaffolder/Manage_Film_Example/api/Manage_Film_Example_db/actor"
	"skaffolder/Manage_Film_Example/api/Manage_Film_Example_db/film"
	"skaffolder/Manage_Film_Example/api/Manage_Film_Example_db/filmmaker"
	"skaffolder/Manage_Film_Example/api/Manage_Film_Example_db/user"

	// end import model

)

func Routes(configuration *config.Config) *chi.Mux {
	router := chi.NewRouter()
	
	// Basic CORS
	cors := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           300,
	})

	router.Use(
		cors.Handler,
		render.SetContentType(render.ContentTypeJSON), // Set content-Type headers as application/json
		middleware.Logger,          // Log API request calls
		middleware.DefaultCompress, // Compress results, mostly gzipping assets and json
		middleware.RedirectSlashes, // Redirect slashes to no slash URL versions
		middleware.Recoverer,       // Recover from panics without crashing server
	)

	// Routing
	router.Route("/api", func(r chi.Router) { 
		r.Mount("/", security.New(configuration).Routes())
		r.Mount("/actor", actor.New(configuration).Routes())
		r.Mount("/film", film.New(configuration).Routes())
		r.Mount("/filmmaker", filmmaker.New(configuration).Routes())
		r.Mount("/user", user.New(configuration).Routes())
	})

	return router
}

func main() {
	configuration, err := config.New()
	if err != nil {
		log.Panicln("Configuration error", err)
	}
	router := Routes(configuration)

	walkFunc := func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		log.Printf("%s %s\n", method, route) // Walk and print out all routes
		return nil
	}
	if err := chi.Walk(router, walkFunc); err != nil {
		log.Panicf("Logging err: %s\n", err.Error()) // panic if there is an error
	}

	// Static filer
	fs := http.FileServer(http.Dir(configuration.Constants.PUBLIC))
	router.Handle("/*", http.StripPrefix("/", fs))

	// Start
	log.Println("Serving application at PORT :" + configuration.Constants.PORT)
	log.Fatal(http.ListenAndServe(":"+configuration.Constants.PORT, router))

}
