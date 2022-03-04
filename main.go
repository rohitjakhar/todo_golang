package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/thedevsaddam/renderer"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var db *mgo.Database
var rnd *renderer.Render

const (
	hostName       string = "localhost:27017"
	dbName         string = "demo_todo"
	collectionName string = "todo"
	port           string = ":9000"
)

type (
	todoModel struct {
		ID        bson.ObjectId `bson: "_id,omitempty"`
		Title     string        `bson: "title"`
		Completed bool          `bson: "completed"`
		CreatedAt time.Time     `bson: "createdAt"`
	}
	todo struct {
		ID        string    `json: "id"`
		Title     string    `json: "title"`
		Completed bool      `json: "completed"`
		CreatedAt time.Time `json: "created_at"`
	}
)

func init() {
	rnd = renderer.New()
	sess, err := mgo.Dial(hostName)
	checkError(err)
	sess.SetMode(mgo.Monotonic, true)

	db = sess.DB(dbName)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	err := rnd.Template(w, http.StatusOK, []string{"static/home.tpl"}, nil)
	checkError(err)
}

func fetchTodos(w http.ResponseWriter, r *http.Request) {
	todos := []todoModel{}
	if err := db.C(collectionName).Find(bson.M{}).All(&todos); err != nil {
		rnd.JSON(w, http.StatusProcessing, renderer.M{
			"message": "Failed to load todo list",
			"error":   err,
		})
		return
	}

	todolist := []todo{}
	for _, t := range todos {
		todolist = append(todolist, todo{
			ID:        t.ID.Hex(),
			Title:     t.Title,
			Completed: t.Completed,
			CreatedAt: t.CreatedAt,
		})
	}
	rnd.JSON(w, http.StatusOK, renderer.M{
		"data": todolist,
	})
}

func addTodod(w http.ResponseWriter, r *http.Request) {
	var t todo
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		rnd.JSON(w, http.StatusProcessing, err)
		return
	}

	if t.Title == "" {
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message": "Title is required",
		})
		return
	}

	tm := todoModel{
		ID:        bson.NewObjectId(),
		Title:     t.Title,
		Completed: false,
		CreatedAt: time.Now(),
	}
	if error := db.C(collectionName).Insert(&tm); error != nil {
		rnd.JSON(w, http.StatusProcessing, renderer.M{
			"message": "Some Errors",
			"error":   error,
		})
		return
	}
	rnd.JSON(w, http.StatusCreated, renderer.M{
		"message": "Todo Added",
		"todo_id": tm.ID.Hex(),
	})
}

func updateTodo(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))

	if !bson.IsObjectIdHex(id) {
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message": "Invalid ID",
		})
		return
	}

	var t todo

	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		rnd.JSON(w, http.StatusProcessing, err)
		return
	}

	if t.Title == "" {
		rnd.JSON(w, http.StatusBadGateway, renderer.M{
			"message": "the Title field is required",
		})
		return
	}

	if err := db.C(collectionName).Update(
		bson.M{"id": bson.ObjectIdHex(id)},
		bson.M{"title": t.Title, "completed": t.Completed},
	); err != nil {
		rnd.JSON(w, http.StatusProcessing, renderer.M{
			"message": "failed to update",
			"error":   err,
		})
		return
	}
}

func deleteTodo(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))

	if !bson.IsObjectIdHex(id) {
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message": "Invalid Id",
		})
		return
	}

	if err := db.C(collectionName).Remove(bson.ObjectIdHex(id)); err != nil {
		rnd.JSON(w, http.StatusProcessing, renderer.M{
			"message": "Failed to delete",
			"error":   err,
		})
		return
	}

	rnd.JSON(w, http.StatusOK, renderer.M{
		"message": "Item Deleted",
	})
}

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", homeHandler)
	r.Mount("/todo", todoHandler())
	log.Println("Run Main...")

	srv := &http.Server{
		Addr:         port,
		Handler:      r,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Println("Listen on port: ", port)
	if err := srv.ListenAndServe(); err != nil {
		log.Println("Error Main...")
		log.Printf("Listen Error: %s", err)
	}
	// go func() {
	// 	log.Println("Listen on port: ", port)
	// 	if err := srv.ListenAndServe(); err != nil {
	// 		log.Println("Error Main...")
	// 		log.Printf("Listen Error: %s", err)
	// 	}
	// }()
}

func todoHandler() http.Handler {
	rg := chi.NewRouter()
	rg.Group(func(r chi.Router) {
		r.Get("/", fetchTodos)
		r.Post("/", addTodod)
		r.Put("/{id}", updateTodo)
		r.Delete("/{id}", deleteTodo)
	})
	return rg
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
