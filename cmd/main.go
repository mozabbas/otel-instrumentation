package main

import (
	"context"
	"log"
	"net/http"

	"instrumentation/internal/tracing"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

var client *mongo.Client

func main() {
	tp, tpErr := tracing.JaegerTraceProvider()
	if tpErr != nil {
		log.Fatal(tpErr)
	}
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	connectMongo()
	setupWebServer()
}

func connectMongo() {
	opts := options.Client()
	opts.Monitor = otelmongo.NewMonitor()
	opts.ApplyURI("mongodb://tracer:admin@localhost:27017")
	client, _ = mongo.Connect(context.Background(), opts)
	//Seed the database with todo's
	docs := []interface{}{
		bson.D{
			{
				"id", "1",
			}, {
				"title", "Buy groceries",
			},
		},
		bson.D{
			{
				"id", "2",
			}, {
				"title", "install Aspecto.io",
			},
		},
		bson.D{
			{
				"id", "3",
			}, {
				"title", "Buy dogz.io domain",
			},
		},
	}
	client.Database("todo").Collection("todos").InsertMany(context.Background(), docs)
}

func setupWebServer() {
	r := gin.Default()
	r.Use(otelgin.Middleware("todo-service"))
	r.GET("/todo", func(c *gin.Context) {
		collection := client.Database("todo").Collection("todos")
		//Important: Make sure to pass c.Request.Context() as the context and not c itself - TBD
		cur, findErr := collection.Find(c.Request.Context(), bson.D{})
		if findErr != nil {
			c.AbortWithError(500, findErr)
			return
		}
		results := make([]interface{}, 0)
		curErr := cur.All(c, &results)
		if curErr != nil {
			c.AbortWithError(500, curErr)
			return
		}
		c.JSON(http.StatusOK, results)
	})
	_ = r.Run(":8080")
}
