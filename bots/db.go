package bots

import (
	"context"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	client     *mongo.Client
	db         *mongo.Database
	collection *mongo.Collection
)

func init() {
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err = mongo.Connect(ctx, options.Client().ApplyURI("mongodb://miner:password@localhost:27017/News?authSource=admin&compressors=disabled&gssapiServiceName=mongodb"))
	if err != nil {
		log.Print(err)
		os.Exit(-1)
	}
	if err = client.Ping(ctx, readpref.Primary()); err != nil {
		log.Println(err)
		os.Exit(-1)
	}
	db = client.Database("News")
	collection = db.Collection("data")
}
