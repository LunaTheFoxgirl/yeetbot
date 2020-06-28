package bot

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const dbDbName string = "yeetbot"
const dbServerCollectionName string = "servers"
const dbUserCollectionName string = "users"

var MongoClient MClient

type MClient struct {
	client *mongo.Client
}

func (self MClient) CountServers() int64 {
	count, err := self.ServersCollection().CountDocuments(context.Background(), bson.D{})
	if err != nil {
		log.Println(err)
		return 0
	}
	return count
}

func (self MClient) ServersCollection() *mongo.Collection {
	return self.client.Database(dbDbName).Collection(dbServerCollectionName)
}

func (self MClient) UsersCollection() *mongo.Collection {
	return self.client.Database(dbDbName).Collection(dbUserCollectionName)
}

func (self MClient) Disconnect() error {
	return self.client.Disconnect(context.Background())
}

func ConnectToDB(connectionString string) error {
	client, err := mongo.NewClient(options.Client().ApplyURI(connectionString))
	if err != nil {
		return err
	}

	// Create a timeout context to make sure we connect within at least 20 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Try to connect to the database
	err = client.Connect(ctx)
	if err != nil {
		return err
	}

	// Perform connection test
	err = client.Ping(ctx, nil)
	if err != nil {
		return err
	}

	// Apply this to our client
	MongoClient = MClient{client}

	log.Println("Connected to database!")
	return nil
}
