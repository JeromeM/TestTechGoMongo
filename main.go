package main

import (
	"github.com/JeromeM/TestTechGoMongo/client"
	"github.com/JeromeM/TestTechGoMongo/config"
	"github.com/JeromeM/TestTechGoMongo/server"
	"github.com/kataras/golog"
)

func main() {
	golog.Infof("Starting TechTest Select.")
	dbURL := config.RequireEnvVar("MONGO_DB_URI")
	dbName := config.RequireEnvVar("MONGO_DB_NAME")
	mongoClient := client.NewMongoClient(dbURL, dbName)

	ServeHTTPRequests(mongoClient)
}
func ServeHTTPRequests(selectDB *client.MongoClient) {
	port := config.RequireEnvVar("API_PORT")
	golog.Infof("Serving API on port %s", port)
	server.NewServer(*selectDB).Serve(port)
}
