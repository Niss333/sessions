package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"syscall"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

type user struct {
	ID        string    `json:"id" bson:"_id"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Email     string    `json:"-"`
	Password  string    `json:"-"`
	Start     time.Time `json:"-"`
}

type sessionStorage struct {
}

type jsonRequest struct {
	UserID string           `json:"user"`
	Type   string           `json:"command"`
	Text   string           `json:"text"`
	From   time.Time        `json:"from"`
	To     time.Time        `json:"to"`
	Data   *json.RawMessage `json:"data"`
}

type jsonReply struct {
	Type   string      `json:"type"`
	Status string      `json:"status"`
	Data   interface{} `json:"data,omitempty"`
}

type appContext struct {
	IP       string
	Port     string
	Path     string
	Server   *http.Server
	Sessions map[string]*user
	Signals  chan os.Signal
}

func main() {
	app := appContext{IP: "172.16.0.6", Port: "8080", Path: "."}

	// Routing
	http.DefaultServeMux.HandleFunc("/static", app.serveRoot)
	http.DefaultServeMux.HandleFunc("/xhr", app.apiHandler)
	http.DefaultServeMux.HandleFunc("/", app.serveRoot)
	app.Server = &http.Server{
		Addr:           fmt.Sprintf("%s:%s", app.IP, app.Port),
		Handler:        http.DefaultServeMux,
		ReadTimeout:    14 * time.Second,
		WriteTimeout:   14 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	fmt.Printf("Starting web/http server on %s...\n", app.Server.Addr)

	// Graceful shutdown
	app.Signals = make(chan os.Signal, 1)
	signal.Notify(app.Signals, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		s := <-app.Signals
		fmt.Printf("RECEIVED SIGNAL: %s\n", s)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		//shutdown the server
		err := app.Server.Shutdown(ctx)
		if err == nil {
			os.Exit(0)
		} else {
			fmt.Printf("Graceful shutdown error: %v\n", err)
			app.Server.Close()
		}
	}()
	fmt.Println(app.Server.ListenAndServe().Error())
}

func (app *appContext) serveRoot(res http.ResponseWriter, req *http.Request) {
	fname := path.Base(req.URL.Path)
	fmt.Printf("[%s] Serving %s for %s\n", time.Now().Truncate(time.Second), fname, req.Header.Get("X-Forwarded-For"))
	res.Header().Set("Cache-Control", "max-age=31536000, immutable")
	res.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeFile(res, req, filepath.Join(app.Path, req.URL.Path))
}

func (app *appContext) apiHandler(response http.ResponseWriter, request *http.Request) {
	//Recover
	// defer func() {
	// 	if err := recover(); err != nil {
	// 		fmt.Println("xhr request failed:", err)
	// 	}
	// }()
	response.Header().Set("Cache-Control", "no-cache")
	response.Header().Set("X-Content-Type-Options", "nosniff")
	response.Header().Set("Content-Type", "application/json; charset=UTF-8")

	var command jsonRequest
	// var params map[string]interface{}
	ctx := context.Background()
	reply := jsonReply{Status: "error", Data: "unimplemented"}

	err := json.NewDecoder(request.Body).Decode(&command)
	if err != nil {
		fmt.Println(err)
		reply.Data = fmt.Sprintf("XHR decoding failed: %v", err)
		json.NewEncoder(response).Encode(reply)
		return
	}
	reply.Type = command.Type
	fmt.Println("Got", command.Type, "command")
	switch command.Type {
	case "login":
	case "logout":
	case "continue":
		if len(command.Text) > 1 {
			result, err := app.slots.DeleteOne(ctx, bson.M{"_id": command.Text})
			if err == nil {
				reply.Status = "ok"
				reply.Data = command.Text
				fmt.Printf("removed %v document(s)\n", result.DeletedCount)
			} else {
				reply.Data = err.Error()
			}

		}
	}
	json.NewEncoder(response).Encode(reply)
}
