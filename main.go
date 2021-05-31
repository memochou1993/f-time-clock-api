package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/joho/godotenv/autoload"
)

const (
	Endpoint = "https://femascloud.com"
)

const (
	ActionTry      = "TRY"
	ActionPunchIn  = "PUNCH_IN"
	ActionPunchOut = "PUNCH_OUT"
)

var (
	logger    *log.Logger
	scheduler = NewScheduler()
	location  = time.FixedZone("UTC+8", 8*60*60)
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
					if err := user.Execute(event.Action); err != nil {
						Log(err.Error())
						continue
					}
					s.Users[userIndex].Events[eventIndex].Success = true
					if err := Notify(user.Email, fmt.Sprint(event.Action)); err != nil {
						Log(err.Error())
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
	r.HandleFunc("/api/attach", Attach).Methods(http.MethodPost)
	r.HandleFunc("/api/detach", Detach).Methods(http.MethodPost)

	log.Fatal(http.ListenAndServe(":80", r))
}

func Attach(w http.ResponseWriter, r *http.Request) {
	var u User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if _, ok := scheduler.Users[u.Credentials.Username]; !ok {
		if err := u.Execute(ActionTry); err != nil {
			Response(w, http.StatusInternalServerError, Payload{Error: err.Error()})
			return
		}
		scheduler.Users[u.Credentials.Username] = u
		Response(w, http.StatusOK, Payload{Data: scheduler.Users[u.Credentials.Username].Events})
		return
	}
	if scheduler.Users[u.Credentials.Username].Credentials.Password != u.Credentials.Password {
		Response(w, http.StatusUnauthorized, Payload{})
		return
	}
	scheduler.Users[u.Credentials.Username] = u
	Response(w, http.StatusOK, Payload{Data: scheduler.Users[u.Credentials.Username].Events})
}

func Detach(w http.ResponseWriter, r *http.Request) {
	var u User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if _, ok := scheduler.Users[u.Credentials.Username]; !ok {
		Response(w, http.StatusNotFound, Payload{})
		return
	}
	if scheduler.Users[u.Credentials.Username].Credentials.Password != u.Credentials.Password {
		Response(w, http.StatusUnauthorized, Payload{})
		return
	}
	delete(scheduler.Users, u.Credentials.Username)
	Response(w, http.StatusOK, Payload{})
}

type User struct {
	Company     string      `json:"company"`
	Cookie      string      `json:"-"`
	Credentials Credentials `json:"credentials"`
	Email       string      `json:"email"`
	Events      []Event     `json:"events"`
}

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Event struct {
	Action     string    `json:"action"`
	Date       time.Time `json:"date"`
	Dispatched bool      `json:"-"`
	Success    bool      `json:"-"`
}

func (u *User) Execute(action string) error {
	if err := u.GetCookie(); err != nil {
		return err
	}
	if err := u.Login(); err != nil {
		return err
	}
	if action == ActionTry {
		if err := u.AddEvent(); err != nil {
			return err
		}
	}
	if action == ActionPunchIn {
		// TODO
	}
	if action == ActionPunchOut {
		// TODO
	}
	if err := u.Logout(); err != nil {
		return err
	}
	return nil
}

func (u *User) GetCookie() error {
	resp, err := http.Get(fmt.Sprintf("%s/%s/%s", Endpoint, u.Company, "/"))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	u.Cookie = resp.Header.Get("Set-Cookie")
	return nil
}

func (u *User) Login() error {
	params := url.Values{}
	params.Add("data[Account][username]", u.Credentials.Username)
	params.Add("data[Account][passwd]", u.Credentials.Password)
	params.Add("data[remember]", `0`)
	body := strings.NewReader(params.Encode())

	return u.Request(http.MethodPost, "accounts/login", body)
}

func (u *User) Logout() error {
	return u.Request(http.MethodPost, "accounts/logout", nil)
}

func (u *User) AddEvent() error {
	params := url.Values{}
	params.Add("data[User][date]", time.Now().In(location).Format("2006-01-02"))
	params.Add("data[UserEvent][start_time][hour]", `08`)
	params.Add("data[UserEvent][start_time][min]", `0`)
	params.Add("data[UserEvent][end_time][hour]", `18`)
	params.Add("data[UserEvent][end_time][min]", `0`)
	params.Add("data[UserEvent][public]", `0`)
	params.Add("data[UserEvent][event]", ``)
	params.Add("data[save]", `確認`)
	body := strings.NewReader(params.Encode())

	return u.Request(http.MethodPost, "users/calendar_event", body)
}

func (u *User) Request(method string, path string, body io.Reader) error {
	url := fmt.Sprintf("%s/%s/%s", Endpoint, u.Company, path)
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Sec-Ch-Ua", "\" Not A;Brand\";v=\"99\", \"Chromium\";v=\"90\", \"Google Chrome\";v=\"90\"")
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Origin", Endpoint)
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Dnt", "1")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.212 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Referer", url)
	req.Header.Set("Accept-Language", "zh-TW,zh;q=0.9,en;q=0.8")
	req.Header.Set("Cookie", u.Cookie)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func Notify(to string, body string) error {
	addr := "smtp.gmail.com:587"
	host := "smtp.gmail.com"
	identity := ""
	from := os.Getenv("SMTP_USERNAME")
	password := os.Getenv("SMTP_PASSWORD")
	subject := "FemasHR Puncher"
	msg := "From:" + from + "\r\n" + "To:" + to + "\r\n" + "Subject:" + subject + "\r\n" + body

	return smtp.SendMail(
		addr,
		smtp.PlainAuth(identity, from, password, host),
		from,
		[]string{to},
		[]byte(msg),
	)
}

type Payload struct {
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

func Response(w http.ResponseWriter, code int, payload Payload) {
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func Log(v interface{}) {
	log.Println(v)
	logger.Println(v)
}
