package database

import (
	"context"
	"fmt"
	"log"
	"time"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)
const (
	hostname       string = "127.0.0.1:27017"
	dbName         string = "project_todo"
	collectionName string = "todo"
	port           string = ":9000"
)

func DBInstance() *mongo.Client {
	ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://"+hostname))
	if err != nil {
		log.Fatal(err)
		defer cancel()
	}
	fmt.Printf("Connection to mongo Successful at %s\n", hostname)
	return client
}

func OpenCollection(client *mongo.Client, collectionName string) *mongo.Collection {
	var collection *mongo.Collection = client.Database("project_todo").Collection(collectionName)
	return collection
}
