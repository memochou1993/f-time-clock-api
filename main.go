package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
	scheduler = NewScheduler()
)

type Scheduler struct {
	Users map[string]User
}

func (s *Scheduler) Start() {
	for range time.Tick(time.Second) {
		// TODO
	}
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		Users: make(map[string]User),
	}
}

type User struct {
	Schedule    []time.Time
	Credentials Credentials
	Email       string
}

type Credentials struct {
	Username string
	Password string
}

func main() {
	go func() {
		scheduler.Start()
	}()

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
		if err := PunchIn(&u.Credentials); err != nil {
			response(w, http.StatusUnauthorized, Payload{Error: err.Error()})
			return
		}
		scheduler.Users[u.Credentials.Username] = u
	}
	if scheduler.Users[u.Credentials.Username].Credentials.Password != u.Credentials.Password {
		response(w, http.StatusUnauthorized, Payload{})
		return
	}
	response(w, http.StatusOK, Payload{Data: scheduler.Users[u.Credentials.Username].Schedule})
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

func PunchIn(c *Credentials) error {
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
