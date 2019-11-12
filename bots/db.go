package bots

import (
	"context"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/x/bsonx"
)

var (
	client *mongo.Client
	// db         *mongo.Database
	// collection *mongo.Collection
)

func init() {
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
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
}

func getDatabaseCollection(db string) (collection *mongo.Collection) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	collection = client.Database(db).Collection(time.Now().Format("02-Jan-06-15:04-MST"))

	c, err := collection.Indexes().List(ctx)
	defer c.Close(ctx)
	key := bson.M{}
	exists := false
	for c.Next(ctx) {
		err = c.Decode(&key)
		if err != nil {
			log.Fatal(err)
		}
		log.Println(key)
		if key["name"] == "code_1" {
			exists = true
		}
	}
	if !exists {
		keys := bsonx.Doc{bsonx.Elem{Key: "code", Value: bsonx.Int32(1)}}
		indexOpts := options.Index().SetUnique(true)
		_, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{Keys: keys, Options: indexOpts})
		if err != nil {
			log.Fatal(err)
		}
	}
	return
}
