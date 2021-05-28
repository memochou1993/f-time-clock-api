package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/tebeka/selenium"
)

const (
	chromeDriverPath = "chromedriver"
	port             = 8080
)

func main() {
	opts := []selenium.ServiceOption{
		selenium.Output(os.Stderr),
	}
	service, err := selenium.NewChromeDriverService(chromeDriverPath, port, opts...)
	if err != nil {
		log.Fatalln(err.Error())
	}
	defer service.Stop()

	caps := selenium.Capabilities{"browserName": "chrome"}
	wd, err := selenium.NewRemote(caps, fmt.Sprintf("http://localhost:%d/wd/hub", port))
	if err != nil {
		log.Fatalln(err.Error())
	}
	defer wd.Quit()

	if err := wd.Get("https://femascloud.com/kklab/accounts/login"); err != nil {
		log.Fatalln(err.Error())
	}

	usernameInput, err := wd.FindElement(selenium.ByCSSSelector, "#user_username")
	if err != nil {
		log.Fatalln(err.Error())
	}
	if err := usernameInput.Clear(); err != nil {
		log.Fatalln(err.Error())
	}
	if err = usernameInput.SendKeys(""); err != nil {
		log.Fatalln(err.Error())
	}

	passwordInput, err := wd.FindElement(selenium.ByCSSSelector, "#user_passwd")
	if err != nil {
		log.Fatalln(err.Error())
	}
	if err := passwordInput.Clear(); err != nil {
		log.Fatalln(err.Error())
	}
	if err = passwordInput.SendKeys(""); err != nil {
		log.Fatalln(err.Error())
	}

	loginButton, err := wd.FindElement(selenium.ByCSSSelector, "#s_buttom")
	if err != nil {
		log.Fatalln(err.Error())
	}
	if err := loginButton.Click(); err != nil {
		log.Fatalln(err.Error())
	}

	menu, err := wd.FindElement(selenium.ByCSSSelector, "#login_user")
	if err != nil {
		log.Fatalln(err.Error())
	}
	if err := menu.Click(); err != nil {
		log.Fatalln(err.Error())
	}

	logoutButton, err := wd.FindElement(selenium.ByCSSSelector, "#login_user li :nth-child(5)")
	if err != nil {
		log.Fatalln(err.Error())
	}
	if err := logoutButton.Click(); err != nil {
		log.Fatalln(err.Error())
	}

	time.Sleep(5 * time.Second)
}
