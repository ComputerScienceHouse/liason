package main

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"log"
	"os"
	"time"
)

var Client = Connect()

var db = os.Getenv("MONGO_DB")
var dbUri = os.Getenv("MONGO_URI")

func Connect() *mongo.Client {

	ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancel()

	opts := options.Client().ApplyURI(dbUri)
	opts.TLSConfig.InsecureSkipVerify = true
	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		log.Panicln(`error connecting to database`, err)
	}

	if err = client.Ping(ctx, readpref.Primary()); err != nil {
		log.Panicln("error pinging database", err)
	}

	log.Println("connected to mongodb")

	return client
}

type Relation struct {
	EboardTS string `bson:eboardTS`
	ClientID string `bson:clientID`
	ClientTS string `bson:clientTS`
}

func LoadRelations() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := Client.Database(db).Collection("relations").Find(ctx, bson.D{})
	if err != nil {
		fmt.Println("HELP", err)
		return
	}
	relations := make([]Relation, 0)

	err = cursor.All(ctx, &relations)
	if err != nil {
		fmt.Println("HELP", err)
		return
	}

	for _, relation := range relations {
		clientString := relation.ClientID + ":" + relation.ClientTS
		EboardRelations[relation.EboardTS] = clientString
		ClientRelations[clientString] = relation.EboardTS
	}

}

func MakeRelation(eboardTS, clientID, clientTS string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	newRelation := Relation{EboardTS: eboardTS, ClientTS: clientTS, ClientID: clientID}

	_, err := Client.Database(db).Collection("relations").InsertOne(ctx, newRelation)
	if err != nil {
		return
	}
}
