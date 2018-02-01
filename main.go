package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"

	"github.com/PuerkitoBio/goquery"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "jobcan"
	app.Usage = "attendance operation command for jobcan"
	app.Version = "0.1.1"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "client_id, c",
			Usage: "client id",
		},
		cli.StringFlag{
			Name:  "login_id, l",
			Usage: "login id",
		},
		cli.StringFlag{
			Name:  "password, p",
			Usage: "password",
		},
		cli.StringFlag{
			Name:  "mode, m",
			Usage: "work_start or work_end",
		},
	}

	app.Action = func(c *cli.Context) error {
		if c.String("client_id") == "" {
			log.Fatal("client_id is required")
		}
		if c.String("login_id") == "" {
			log.Fatal("login_id is required")
		}
		if c.String("password") == "" {
			log.Fatal("password is required")
		}
		if c.String("mode") == "" {
			log.Fatal("mode is required. work_start or work_end")
		}

		jar, err := cookiejar.New(nil)
		if err != nil {
			log.Fatal(err)
		}

		client := &http.Client{Jar: jar}
		login(client, c.String("client_id"), c.String("login_id"), c.String("password"))
		token, group_id := fetchTokenAndGroup(client)
		pushDakoku(client, c.String("mode"), token, group_id)

		fmt.Println("done!")
		fmt.Println("see https://ssl.jobcan.jp/employee/")
		return nil
	}

	app.Run(os.Args)

}

func login(client *http.Client, client_id, login_id, password string) {
	values := url.Values{}
	values.Add("client_login_id", client_id)
	values.Add("client_manager_login_id", login_id)
	values.Add("client_login_password", password)
	values.Add("login_type", "2")
	values.Add("url", "https://ssl.jobcan.jp/client/")
	res, err := client.PostForm("https://ssl.jobcan.jp/login/client", values)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatal("Login error StatusCode=" + string(res.StatusCode))
	}
	employeeLogin(client)
}

func fetchEmployeeCode(client *http.Client) string {
	res, err := client.Get("https://ssl.jobcan.jp/client")
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatal("Login error StatusCode=" + string(res.StatusCode))
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	attr, _ := doc.Find("#rollover-menu > li:nth-child(2)").Attr("onclick")
	str := []byte(attr)
	assigned := regexp.MustCompile("code=([0-9a-f]+)")
	group := assigned.FindSubmatch(str)
	return string(group[1])
}

func employeeLogin(client *http.Client) {
	code := fetchEmployeeCode(client)
	res, err := client.Get("https://ssl.jobcan.jp/login/pc-employee/try?code=" + code)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatal("Login error StatusCode=" + string(res.StatusCode))
	}
}

func pushDakoku(client *http.Client, mode string, token string, group_id string) {
	values := url.Values{}
	values.Add("is_yakin", "0")
	values.Add("adit_item", mode)
	values.Add("notice", "")
	values.Add("token", token)
	values.Add("adit_group_id", group_id)
	res, err := client.PostForm("https://ssl.jobcan.jp/employee/index/adit", values)
	if err != nil {
		log.Fatal(err)
	}

	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatal("Post error StatusCode=" + string(res.StatusCode))
		return
	}
}

func fetchTokenAndGroup(client *http.Client) (string, string) {
	res, err := client.Get("https://ssl.jobcan.jp/employee")
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	doc, _ := goquery.NewDocumentFromReader(res.Body)
	token, _ := doc.Find("input[name='token']").Attr("value")
	group_id, _ := doc.Find("select#adit_group_id option:first-child").Attr("value")
	return token, group_id
}
