package main

import (
	"fmt"
	"gallerio/utils/errors"
	"gallerio/utils/rand"
	"github.com/gorilla/csrf"
	"log"
	"net/http"
	
	"gallerio/controllers"
	"gallerio/middlewares"
	"gallerio/models"
	
	"github.com/gorilla/mux"
)


func main() {
	cfg := DefaultConfig()
	dbCfg := DefaultPostgresConfig()
	services, err := models.NewServices(
		models.WithGorm(dbCfg.Dialect(), dbCfg.ConnectionInfo()),
		models.WithLogMode(cfg.IsDevelopment()),
		models.WithUser(cfg.Pepper, cfg.HMACKey),
		models.WithGallery(),
		models.WithImage(),
	)
	if err != nil {
		panic(err)
	}
	defer services.Close()
	services.AutoMigrate()

	router := mux.NewRouter()
	usersController := controllers.NewUsersController(services.User)
	galleriesController := controllers.NewGalleriesController(services.Gallery, services.Image, router)
	coreController := controllers.NewStaticController()
	
	b, err := rand.Bytes(32)
	errors.Must(err)
	csrfMw := csrf.Protect(b, csrf.Secure(cfg.IsProduction()))
	assignUserMw := middlewares.AssignUser{
		UserService: services.User,
	}
	loginRequiredMw := middlewares.LoginRequired{
		UserService: services.User,
	}

	// Static Routes
	router.Handle("/", coreController.HomeView).Methods("GET")
	router.Handle("/contact", coreController.ContactView).Methods("GET")

	// Accounts Routes
	router.Handle("/signin", usersController.SignInView).Methods("GET")
	router.HandleFunc("/signin", usersController.SignIn).Methods("POST")
	router.Handle("/signup", usersController.SignUpView).Methods("GET")
	router.HandleFunc("/signup", usersController.SignUp).Methods("POST")

	// Galleries Routes
	router.Handle("/galleries/new",
		loginRequiredMw.Apply(galleriesController.New)).Methods("GET")
	router.HandleFunc("/galleries",
		loginRequiredMw.ApplyFunc(galleriesController.Index)).Methods("GET")
	router.HandleFunc("/galleries",
		loginRequiredMw.ApplyFunc(galleriesController.Create)).Methods("POST")
	router.HandleFunc("/galleries/{id:[0-9]+}",
		galleriesController.Show).Methods("GET").Name(controllers.ShowGalleryName)
	router.HandleFunc("/galleries/{id:[0-9]+}/edit",
		loginRequiredMw.ApplyFunc(galleriesController.Edit)).
		Methods("GET").Name(controllers.EditGalleryName)
	router.HandleFunc("/galleries/{id:[0-9]+}/update",
		loginRequiredMw.ApplyFunc(galleriesController.Update)).Methods("POST")
	router.HandleFunc("/galleries/{id:[0-9]+}/delete",
		loginRequiredMw.ApplyFunc(galleriesController.Delete)).Methods("POST")
	router.HandleFunc("/galleries/{id:[0-9]+}/images",
		loginRequiredMw.ApplyFunc(galleriesController.UploadImage)).Methods("POST")
	router.HandleFunc("/galleries/{id:[0-9]+}/images/{filename}/delete",
		loginRequiredMw.ApplyFunc(galleriesController.DeleteImage)).Methods("POST")
	
	// Media Routes
	mediaHandler := http.FileServer(http.Dir("./media/"))
	router.PathPrefix("/media/").Handler(http.StripPrefix("/media/", mediaHandler))
	
	// Static Routes
	staticHandler := http.FileServer(http.Dir("./static/"))
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", staticHandler))

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", cfg.Port), csrfMw(assignUserMw.Apply(router))))
}
