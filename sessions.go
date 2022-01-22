package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gorilla/securecookie"
)

type user struct {
	ID        string    `json:"id" bson:"_id"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Email     string    `json:"-"`
	Password  string    `json:"-"`
	Start     time.Time `json:"-"`
}

type jsonRequest struct {
	UserID string           `json:"user"`
	Type   string           `json:"command"`
	Text   string           `json:"text"`
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
	Signals  chan os.Signal
	Sessions map[string]*user
	Secretly *securecookie.SecureCookie
}

func main() {
	powerUser := user{ID: "alpha", Email: "alef@app.com", Password: "omega", FirstName: "Дорогой", LastName: "Пользователь"}
	hashKey := securecookie.GenerateRandomKey(32)
	blockKey := securecookie.GenerateRandomKey(32)

	app := appContext{IP: "127.0.0.1", Port: "8080", Path: "."}
	app.Sessions = make(map[string]*user)
	app.Sessions[powerUser.ID] = &powerUser
	app.Secretly = securecookie.New(hashKey, blockKey)

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
	fmt.Println(req.URL.Path)
	// overkill
	req.Header.Del("If-Modified-Since")
	res.Header().Set("Cache-Control", "no-cache, private, max-age=0")
	res.Header().Set("Expires", time.Unix(0, 0).Format(http.TimeFormat))
	res.Header().Set("Pragma", "no-cache")
	res.Header().Set("X-Accel-Expires", "0")
	res.Header().Set("X-Content-Type-Options", "nosniff")
	if req.URL.Path == "/welcome.html" {
		if _, ok := app.authenticate(req); ok {
			http.ServeFile(res, req, filepath.Join(app.Path, "welcome.html"))
		} else {
			http.ServeFile(res, req, filepath.Join(app.Path, "error404.html"))
		}
		return
	}
	http.ServeFile(res, req, filepath.Join(app.Path, req.URL.Path))
}

func (app *appContext) authenticate(request *http.Request) (string, bool) {
	if cookie, err := request.Cookie("session"); err == nil {
		cookieData := make(map[string]string)
		if err = app.Secretly.Decode("session", cookie.Value, &cookieData); err == nil {
			user, found := app.Sessions[cookieData["id"]]
			if found && time.Now().Before(user.Start.Add(5*time.Minute)) {
				fmt.Println("Request authenticated for user", user.ID)
				return user.ID, true
			}
		}
	}
	fmt.Println("Request not authenticated")
	return "none", false
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
	reply := jsonReply{Status: "error", Data: "unimplemented"}

	err := json.NewDecoder(request.Body).Decode(&command)
	if err != nil {
		reply.Data = fmt.Sprintf("XHR decoding failed: %v", err)
		json.NewEncoder(response).Encode(reply)
		return
	}
	reply.Type = command.Type
	fmt.Println("Got", command.Type, "command")
	switch command.Type {
	case "login":
		user, found := app.Sessions[command.UserID]
		if found {
			if command.Text == user.Password {
				cookieData := map[string]string{"id": user.ID}
				if encrypted, err := app.Secretly.Encode("session", cookieData); err == nil {
					cookie := &http.Cookie{
						Path:     "/",
						Name:     "session",
						Value:    encrypted,
						Expires:  time.Now().Add(5 * time.Minute),
						Secure:   false,
						HttpOnly: true,
					}
					http.SetCookie(response, cookie)
				}
				fmt.Println("User [", user.ID, "] logged in")
				user.Start = time.Now()
				reply.Status = "ok"
			} else {
				reply.Data = "Entered password doesn't match"
			}
		} else {
			reply.Data = "Entered id doesn't exist"
		}
	case "logout":
		cookie := http.Cookie{Name: "session", Value: "", Path: "/", MaxAge: -1}
		http.SetCookie(response, &cookie)
		reply.Status = "ok"
	case "continue":
		if id, ok := app.authenticate(request); ok {
			user := app.Sessions[id]
			expiration := time.Until(user.Start.Add(5 * time.Minute)).Milliseconds()
			reply.Data = map[string]interface{}{"firstName": user.FirstName, "lastName": user.LastName, "sessionExpiresIn": expiration}
		} else {
			reply.Data = "denied"
		}
		reply.Status = "ok"
	}
	json.NewEncoder(response).Encode(reply)
}
