package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
)

const (
	chromeDriverPath = "./chromedriver"
	url              = "https://femascloud.com/kklab/accounts/login"
	port             = 8080
)

var (
	logger    *log.Logger
	scheduler = NewScheduler()
)

type Scheduler struct {
	Users map[string]User
}

func (s *Scheduler) Start() {
	for range time.Tick(time.Minute) {
		for userIndex, user := range s.Users {
			for eventIndex, event := range user.Events {
				duration := time.Now().Sub(event.Date).Seconds()
				if !event.Dispatched && duration >= 0 && duration < 60 {
					s.Users[userIndex].Events[eventIndex].Dispatched = true
					if event.Action == "IN" {
						if err := Punch(&user.Credentials, &Script{In: true}); err != nil {
							logger.Println(err.Error())
							return
						}
						s.Users[userIndex].Events[eventIndex].Success = true
						logger.Printf("%s: IN", user.Credentials.Username)
					}
					if event.Action == "OUT" {
						if err := Punch(&user.Credentials, &Script{Out: true}); err != nil {
							logger.Println(err.Error())
							return
						}
						s.Users[userIndex].Events[eventIndex].Success = true
						logger.Printf("%s: OUT", user.Credentials.Username)
					}
				}
			}
		}
	}
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		Users: make(map[string]User),
	}
}

type User struct {
	Credentials Credentials `json:"credentials"`
	Email       string      `json:"email"`
	Events      []Event     `json:"events"`
}

type Event struct {
	Action     string    `json:"action"`
	Date       time.Time `json:"date"`
	Dispatched bool      `json:"-"`
	Success    bool      `json:"-"`
}

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Script struct {
	In  bool
	Out bool
}

func init() {
	file, err := os.OpenFile("./logs/log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	logger = log.New(file, "", log.Ldate|log.Ltime)
}

func main() {
	go scheduler.Start()

	r := mux.NewRouter()
	r.HandleFunc("/attach", AttachHandler).Methods(http.MethodPost)
	r.HandleFunc("/detach", DetachHandler).Methods(http.MethodPost)
	log.Fatal(http.ListenAndServe(":8000", r))
}

func AttachHandler(w http.ResponseWriter, r *http.Request) {
	var u User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if _, ok := scheduler.Users[u.Credentials.Username]; !ok {
		if err := Punch(&u.Credentials, &Script{}); err != nil {
			logger.Println(err.Error())
			response(w, http.StatusUnauthorized, Payload{Error: err.Error()})
			return
		}
		scheduler.Users[u.Credentials.Username] = u
		response(w, http.StatusOK, Payload{Data: scheduler.Users[u.Credentials.Username].Events})
		return
	}
	if scheduler.Users[u.Credentials.Username].Credentials.Password != u.Credentials.Password {
		response(w, http.StatusUnauthorized, Payload{})
		return
	}
	scheduler.Users[u.Credentials.Username] = u
	response(w, http.StatusOK, Payload{Data: scheduler.Users[u.Credentials.Username].Events})
}

func DetachHandler(w http.ResponseWriter, r *http.Request) {
	var u User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if _, ok := scheduler.Users[u.Credentials.Username]; !ok {
		response(w, http.StatusNotFound, Payload{})
		return
	}
	if scheduler.Users[u.Credentials.Username].Credentials.Password != u.Credentials.Password {
		response(w, http.StatusUnauthorized, Payload{})
		return
	}
	delete(scheduler.Users, u.Credentials.Username)
	response(w, http.StatusOK, Payload{})
}

func Punch(c *Credentials, s *Script) error {
	opts := []selenium.ServiceOption{}
	service, err := selenium.NewChromeDriverService(chromeDriverPath, port, opts...)
	if err != nil {
		return err
	}
	defer service.Stop()

	caps := selenium.Capabilities{"browserName": "chrome"}
	chromeCaps := chrome.Capabilities{
		Args: []string{
			"--headless",
		},
	}
	caps.AddChrome(chromeCaps)
	wd, err := selenium.NewRemote(caps, fmt.Sprintf("http://localhost:%d/wd/hub", port))
	if err != nil {
		return err
	}
	defer wd.Quit()

	if err := wd.Get(url); err != nil {
		return err
	}

	usernameInput, err := wd.FindElement(selenium.ByCSSSelector, "#user_username")
	if err != nil {
		return err
	}
	if err := usernameInput.Clear(); err != nil {
		return err
	}
	if err = usernameInput.SendKeys(c.Username); err != nil {
		return err
	}

	passwordInput, err := wd.FindElement(selenium.ByCSSSelector, "#user_passwd")
	if err != nil {
		return err
	}
	if err := passwordInput.Clear(); err != nil {
		return err
	}
	if err = passwordInput.SendKeys(c.Password); err != nil {
		return err
	}

	loginButton, err := wd.FindElement(selenium.ByCSSSelector, "#s_buttom")
	if err != nil {
		return err
	}
	if err := loginButton.Click(); err != nil {
		return err
	}

	menu, err := wd.FindElement(selenium.ByCSSSelector, "#login_user")
	if err != nil {
		return err
	}
	if err := menu.Click(); err != nil {
		return err
	}

	logoutButton, err := wd.FindElement(selenium.ByCSSSelector, "#login_user li :nth-child(5)")
	if err != nil {
		return err
	}
	if err := logoutButton.Click(); err != nil {
		return err
	}

	return nil
}

type Payload struct {
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

func response(w http.ResponseWriter, code int, payload Payload) {
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
