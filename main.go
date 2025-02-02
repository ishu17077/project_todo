package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	middleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
	"github.com/ishu17077/project_todo/database"
	"github.com/thedevsaddam/renderer"
	"go.mongodb.org/mongo-driver/bson"
	primitive "go.mongodb.org/mongo-driver/bson/primitive"
	mongo "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var rnd *renderer.Render
var collection *mongo.Collection
var validate = validator.New()

const (
	hostname       string = "127.0.0.1:27017"
	dbName         string = "project_todo"
	collectionName string = "todo"
	port           string = ":9000"
)

type (
	todoModel struct {
		ID          primitive.ObjectID `bson:"_id"`
		Title       string             `json:"title"`
		IsCompleted bool               `json:"is_completed" validate:"required"`
		CreatedAt   time.Time          `json:"created_at" validate:"required"`
		UpdatedAt   time.Time          `json:"updated_at"`
	}
	todo struct {
		ID          string    `json:"_id"`
		Title       string    `json:"title"`
		IsCompleted bool      `json:"is_completed"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
	}
)

func init() {
	rnd = renderer.New()
	var client *mongo.Client = database.DBInstance()
	collection = database.OpenCollection(client, collectionName)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	err := rnd.Template(w, http.StatusOK, []string{"./static/home.html"}, nil)
	checkErr(err)
}

func main() {
	stopChannel := make(chan os.Signal)
	signal.Notify(stopChannel, os.Interrupt)
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", homeHandler)
	r.Mount("/todo", todoHandlers())

	srv := &http.Server{
		Addr:         port,
		Handler:      r,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	/**
	*? go func executes the function in a separate goroutine.
	*? It's likely that the reason you are not seeing it print anything is that the program is finishing and exiting prior to the print command from that call being executed.
	*? If you want to guarantee that goroutines finish, you should look up WaitGroups in the sync package.
	 */
	go func() {
		log.Println("Listening on port ", port)
		if err := srv.ListenAndServe(); err != nil {
			log.Printf("listen:%s\n", err)
		}
	}()
	<-stopChannel
	log.Println("Shutting down server....")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	srv.Shutdown(ctx)
	defer cancel()
	log.Println("Server Gracefully shut down")

}

func todoHandlers() http.Handler {
	rg := chi.NewRouter()
	rg.Group(func(r chi.Router) {
		r.Get("/", fetchTodos)
		r.Post("/", createTodo)
		r.Put("/{id}", updateTodo)
		r.Delete("/{id}", deleteTodo)
	})
	return rg
}

func fetchTodos(w http.ResponseWriter, r *http.Request) {
	var ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	res, err := collection.Find(ctx, bson.M{})
	todos := []todoModel{}
	if err != nil {
		rnd.JSON(w, http.StatusInternalServerError, renderer.M{
			"message": "Failed to fetch todo",
			"error":   err,
		})
		defer cancel()
		return
	}
	if err := res.All(ctx, &todos); err != nil {
		defer cancel()
		log.Fatal(err)
		return
	}
	todoList := []todo{}
	for _, t := range todos {
		todoList = append(todoList, todo{
			ID:          t.ID.Hex(),
			Title:       t.Title,
			IsCompleted: t.IsCompleted,
			CreatedAt:   t.CreatedAt,
			UpdatedAt:   t.UpdatedAt,
		})
	}
	defer cancel()
	rnd.JSON(w, http.StatusOK, renderer.M{
		"data": todoList,
	})
}

func createTodo(w http.ResponseWriter, r *http.Request) {
	var ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	var t todo
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		rnd.JSON(w, http.StatusBadRequest, err)
		defer cancel()
		return
	}
	validationErr := validate.Struct(&t)
	if validationErr != nil {
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message": "Error parsing your request",
			"error":   validationErr,
		})
		defer cancel()
		return
	}
	if t.Title == "" {
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message": "Title is required",
		})
		defer cancel()
		return
	}
	todoModel := todoModel{
		ID:          primitive.NewObjectID(),
		Title:       t.Title,
		IsCompleted: false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	result, insertErr := collection.InsertOne(ctx, todoModel)
	if insertErr != nil {
		defer cancel()
		rnd.JSON(w, http.StatusInternalServerError, renderer.M{
			"message": "Todo Creation failed",
			"error":   insertErr,
		})
		return
	}
	defer cancel()
	rnd.JSON(w, http.StatusCreated, renderer.M{
		"message": "Todo creation successful",
		"result":  result,
		"todo_id": todoModel.ID.Hex(),
	})
}

func deleteTodo(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	var ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	objectId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		log.Panic(id)
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message": "Error Parsing your request",
			"error":   err,
		})
		defer cancel()
		return
	}
	filter := bson.M{"_id": objectId}
	res, deleteErr := collection.DeleteOne(ctx, filter)
	if deleteErr != nil {
		rnd.JSON(w, http.StatusInternalServerError, renderer.M{
			"message": "Error deleting the todo",
			"error":   deleteErr,
		})
		defer cancel()
		return
	}
	defer cancel()
	rnd.JSON(w, http.StatusOK, renderer.M{
		"message": "Todo deletion successful",
		"todo_id": id,
		"result":  res,
	})
}

func updateTodo(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	fmt.Print(id)
	var ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		defer cancel()
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message": "Error Parsing your request",
			"error":   err,
		})
		return
	}

	var todo todo
	if err := json.NewDecoder(r.Body).Decode(&todo); err != nil {
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message": "Bad request",
		})
	}
	var updateObj primitive.D

	if todo.Title != "" || &(todo.Title) != nil {
		updateObj = append(updateObj, bson.E{Key: "title", Value: todo.Title})
		todo.UpdatedAt, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		updateObj = append(updateObj, bson.E{Key: "updated_at", Value: todo.UpdatedAt})
		filter := bson.M{"_id": objectID}
		upsert := true
		opts := options.UpdateOptions{
			Upsert: &upsert,
		}
		result, err := collection.UpdateOne(ctx, filter, bson.D{
			{Key: "$set", Value: updateObj},
		}, &opts)
		if err != nil {
			rnd.JSON(w, http.StatusInternalServerError, renderer.M{
				"message": "Update Failed",
				"error":   err,
			})
			defer cancel()
			return
		}
		defer cancel()
		rnd.JSON(w, http.StatusOK, renderer.M{
			"message": "Update Successful",
			"todo_id": id,
			"result":  result,
		})
	} else {
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message": "Title is required",
		})
		defer cancel()
	}
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
