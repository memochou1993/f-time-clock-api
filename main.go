package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"strings"
	"time"

	_ "github.com/joho/godotenv/autoload"
)

const (
	Scheme = "https"
	Host   = "femascloud.com"
)

const (
	ActionIssueToken = "ISSUE_TOKEN"
	ActionPunchIn    = "PUNCH_IN"
	ActionPunchOut   = "PUNCH_OUT"
)

var (
	logger    *log.Logger
	scheduler = NewScheduler()
	location  = time.FixedZone("UTC+8", 8*60*60)
)

func init() {
	file, err := os.OpenFile("./logs/log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	logger = log.New(file, "", log.Ldate|log.Ltime)
}

func main() {
	go scheduler.Start()
	go scheduler.Prune()

	r := mux.NewRouter()
	r.HandleFunc("/api/attach", Attach).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/api/detach", Detach).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/api/verify", Verify).Methods(http.MethodPost, http.MethodOptions)

	log.Fatal(http.ListenAndServe(":80", r))
}

func Attach(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		Response(w, http.StatusOK, nil)
		return
	}
	u := NewUser()
	if err := json.NewDecoder(r.Body).Decode(u); err != nil {
		Response(w, http.StatusBadRequest, Payload{Error: err.Error()})
		return
	}
	if u.Credentials == nil {
		Response(w, http.StatusBadRequest, nil)
		return
	}
	if u.ID == "" || u.Company == "" || u.Credentials.Username == "" {
		Response(w, http.StatusBadRequest, nil)
		return
	}
	if _, ok := scheduler.Users[u.Credentials.Username]; !ok {
		if u.Credentials.Password == "" {
			Response(w, http.StatusBadRequest, nil)
			return
		}
		if err := u.Execute(ActionIssueToken); err != nil {
			Response(w, http.StatusInternalServerError, Payload{Error: err.Error()})
			return
		}
		scheduler.Users[u.Credentials.Username] = u
		Response(w, http.StatusCreated, nil)
		return
	}
	if !scheduler.Users[u.Credentials.Username].Verified {
		Response(w, http.StatusForbidden, nil)
		return
	}
	if scheduler.Users[u.Credentials.Username].Token != u.Token {
		Response(w, http.StatusUnauthorized, nil)
		return
	}
	scheduler.Users[u.Credentials.Username].Email = u.Email
	scheduler.Users[u.Credentials.Username].Events = u.Events
	Response(w, http.StatusOK, Payload{Data: User{Events: u.Events}})
}

func Detach(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		Response(w, http.StatusOK, nil)
		return
	}
	u := NewUser()
	if err := json.NewDecoder(r.Body).Decode(u); err != nil {
		Response(w, http.StatusBadRequest, Payload{Error: err.Error()})
		return
	}
	if u.Credentials == nil {
		Response(w, http.StatusBadRequest, nil)
		return
	}
	if u.Credentials.Username == "" {
		Response(w, http.StatusBadRequest, nil)
		return
	}
	if _, ok := scheduler.Users[u.Credentials.Username]; !ok {
		Response(w, http.StatusNotFound, nil)
		return
	}
	if !scheduler.Users[u.Credentials.Username].Verified {
		Response(w, http.StatusForbidden, nil)
		return
	}
	if scheduler.Users[u.Credentials.Username].Token == u.Token {
		delete(scheduler.Users, u.Credentials.Username)
		Response(w, http.StatusNoContent, nil)
		return
	}
	if scheduler.Users[u.Credentials.Username].Credentials.Password == u.Credentials.Password {
		delete(scheduler.Users, u.Credentials.Username)
		Response(w, http.StatusNoContent, nil)
		return
	}
	Response(w, http.StatusUnauthorized, nil)
}

func Verify(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		Response(w, http.StatusOK, nil)
		return
	}
	u := NewUser()
	if err := json.NewDecoder(r.Body).Decode(u); err != nil {
		Response(w, http.StatusBadRequest, Payload{Error: err.Error()})
		return
	}
	if u.Credentials == nil {
		Response(w, http.StatusBadRequest, nil)
		return
	}
	if u.Credentials.Username == "" || u.Token == "" {
		Response(w, http.StatusBadRequest, nil)
		return
	}
	if _, ok := scheduler.Users[u.Credentials.Username]; !ok {
		Response(w, http.StatusNotFound, nil)
		return
	}
	if scheduler.Users[u.Credentials.Username].Token != u.Token {
		Response(w, http.StatusUnauthorized, nil)
		return
	}
	scheduler.Users[u.Credentials.Username].Verified = true
	Response(w, http.StatusOK, nil)
}

type Scheduler struct {
	Users map[string]*User
}

func (s *Scheduler) Start() {
	for range time.Tick(time.Minute) {
		for ui, user := range s.Users {
			if !user.Verified {
				continue
			}
			for ei, event := range user.Events {
				diff := time.Now().Sub(event.Date)
				if !event.Dispatched && diff >= 0 && diff < time.Minute {
					s.Users[ui].Events[ei].Dispatched = true
					if err := user.Execute(event.Action); err != nil {
						Log(err.Error())
						go Notify(user.Email, fmt.Sprintf("Error: %s", err.Error()))
						continue
					}
				}
			}
		}
	}
}

func (s *Scheduler) Prune() {
	for range time.Tick(time.Minute) {
		for _, user := range s.Users {
			if !user.Verified && time.Now().Sub(user.CreatedAt) > 5*time.Minute {
				delete(scheduler.Users, user.Credentials.Username)
			}
		}
	}
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		Users: make(map[string]*User),
	}
}

type User struct {
	ID          string       `json:"id,omitempty"`
	Company     string       `json:"company,omitempty"`
	Cookie      string       `json:"-"`
	Credentials *Credentials `json:"credentials,omitempty"`
	Email       string       `json:"email,omitempty"`
	Events      []Event      `json:"events"`
	Token       string       `json:"token,omitempty"`
	Verified    bool         `json:"-"`
	CreatedAt   time.Time    `json:"-"`
}

func NewUser() *User {
	return &User{
		CreatedAt: time.Now(),
	}
}

type Credentials struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type Event struct {
	Action     string    `json:"action,omitempty"`
	Date       time.Time `json:"date,omitempty"`
	Dispatched bool      `json:"-"`
}

func (u *User) Execute(action string) error {
	if err := u.SetCookie(); err != nil {
		return err
	}
	if err := u.Login(); err != nil {
		return err
	}
	switch action {
	case ActionIssueToken:
		u.Token = NewToken()
		if err := u.CreateEvent(u.Token); err != nil {
			return err
		}
		go Notify(u.Email, fmt.Sprintf("A new token has been issued. Please check your calendar."))
	case ActionPunchIn:
		if err := u.PunchIn(); err != nil {
			return err
		}
		if err := u.ListStatus(); err != nil {
			return err
		}
		go Notify(u.Email, fmt.Sprintf("Punched in successfully!"))
	case ActionPunchOut:
		if err := u.PunchOut(); err != nil {
			return err
		}
		if err := u.ListStatus(); err != nil {
			return err
		}
		go Notify(u.Email, fmt.Sprintf("Punched out successfully!"))
	}
	if err := u.Logout(); err != nil {
		return err
	}
	return nil
}

func (u *User) SetCookie() error {
	resp, err := http.Get(fmt.Sprintf("%s://%s/%s/", Scheme, Host, u.Company))
	if err != nil {
		return err
	}
	defer CloseBody(resp.Body)
	u.Cookie = resp.Header.Get("Set-Cookie")
	return nil
}

func (u *User) Login() error {
	params := url.Values{}
	params.Add("data[Account][username]", u.Credentials.Username)
	params.Add("data[Account][passwd]", u.Credentials.Password)
	params.Add("data[remember]", `0`)
	body := strings.NewReader(params.Encode())

	return u.Request("accounts/login", body)
}

func (u *User) Logout() error {
	return u.Request("accounts/logout", nil)
}

func (u *User) PunchIn() error {
	params := url.Values{}
	params.Add("_method", `POST`)
	params.Add("data[ClockRecord][user_id]", u.ID)
	params.Add("data[AttRecord][user_id]", u.ID)
	params.Add("data[ClockRecord][shift_id]", `2`)
	params.Add("data[ClockRecord][period]", `1`)
	params.Add("data[ClockRecord][clock_type]", `S`)
	params.Add("data[ClockRecord][latitude]", ``)
	params.Add("data[ClockRecord][longitude]", ``)
	body := strings.NewReader(params.Encode())

	return u.Request("users/clock_listing", body)
}

func (u *User) PunchOut() error {
	params := url.Values{}
	params.Add("_method", `POST`)
	params.Add("data[ClockRecord][user_id]", u.ID)
	params.Add("data[AttRecord][user_id]", u.ID)
	params.Add("data[ClockRecord][shift_id]", `2`)
	params.Add("data[ClockRecord][period]", `1`)
	params.Add("data[ClockRecord][clock_type]", `E`)
	params.Add("data[ClockRecord][latitude]", ``)
	params.Add("data[ClockRecord][longitude]", ``)
	body := strings.NewReader(params.Encode())

	return u.Request("users/clock_listing", body)
}

func (u *User) CreateEvent(event string) error {
	params := url.Values{}
	params.Add("_method", `POST`)
	params.Add("data[User][date]", time.Now().In(location).Format("2006-01-02"))
	params.Add("data[UserEvent][start_time][hour]", `08`)
	params.Add("data[UserEvent][start_time][min]", `0`)
	params.Add("data[UserEvent][end_time][hour]", `18`)
	params.Add("data[UserEvent][end_time][min]", `0`)
	params.Add("data[UserEvent][public]", `0`)
	params.Add("data[UserEvent][event]", event)
	params.Add("data[save]", `確認`)
	body := strings.NewReader(params.Encode())

	return u.Request("users/calendar_event", body)
}

func (u *User) ListStatus() error {
	return u.Request("users/att_status_listing", nil)
}

func (u *User) Request(path string, body io.Reader) error {
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s://%s/%s/%s", Scheme, Host, u.Company, path), body)
	if err != nil {
		return err
	}
	req.Host = Host
	req.Header.Set("Accept", "text/javascript, text/html, application/xml, text/xml, */*")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Accept-Language", "zh-TW,zh;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", u.Cookie)
	req.Header.Set("Origin", fmt.Sprintf("%s://%s", Scheme, Host))
	req.Header.Set("Referer", fmt.Sprintf("%s://%s/%s/users/main", Scheme, Host, u.Company))
	req.Header.Set("Sec-Ch-Ua", "\" Not;A Brand\";v=\"99\", \"Google Chrome\";v=\"91\", \"Chromium\";v=\"91\"")
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.77 Safari/537.36")
	req.Header.Set("X-Prototype-Version", "1.7")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer CloseBody(resp.Body)

	return nil
}

func Notify(to string, body string) {
	if to == "" || body == "" {
		return
	}

	addr := "smtp.gmail.com:587"
	host := "smtp.gmail.com"
	identity := ""
	from := os.Getenv("SMTP_USERNAME")
	password := os.Getenv("SMTP_PASSWORD")
	subject := "FemasHR Puncher"
	auth := smtp.PlainAuth(identity, from, password, host)

	header := make(map[string]string)
	header["From"] = from
	header["To"] = to
	header["Subject"] = subject
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "text/plain; charset=\"utf-8\""
	header["Content-Transfer-Encoding"] = "base64"
	message := ""
	for k, v := range header {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += base64.StdEncoding.EncodeToString([]byte("\r\n" + body + "\r\n"))

	if err := smtp.SendMail(addr, auth, from, []string{to}, []byte(message)); err != nil {
		Log(err.Error())
	}
}

type Payload struct {
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

func Response(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.WriteHeader(code)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func NewToken() string {
	rand.Seed(time.Now().Unix())
	token := ""
	for i := 0; i < 6; i++ {
		token += string('A' + rune(rand.Intn(26)))
	}
	return token
}

func CloseBody(closer io.ReadCloser) {
	if err := closer.Close(); err != nil {
		log.Fatal(err.Error())
	}
}

func Log(v interface{}) {
	log.Println(v)
	logger.Println(v)
}
