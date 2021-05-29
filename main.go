package main

import (
	"bufio"
	"fmt"
	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
	"golang.org/x/crypto/ssh/terminal"
	"log"
	"os"
	"strings"
	"syscall"
)

const (
	chromeDriverPath = "./chromedriver"
	url              = "https://femascloud.com/kklab/accounts/login"
	port             = 8080
)

type Credentials struct {
	Username string
	Password string
}

func main() {
	c, err := NewCredentials()
	if err != nil {
		log.Println(err.Error())
	}
	if err := PunchIn(c); err != nil {
		log.Println("Incorrect username or password.")
	}
}

func NewCredentials() (*Credentials, error) {
	reader := bufio.NewReader(os.Stdin)

	c := &Credentials{}

	fmt.Print("Enter Username: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	c.Username = strings.TrimSpace(username)

	fmt.Print("Enter Password: ")
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return nil, err
	}
	c.Password = string(bytePassword)

	fmt.Print("\n")

	return c, nil
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
