package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/user"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/PuerkitoBio/goquery"
	"github.com/Songmu/prompter"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
)

// Config is command parameters
type Config struct {
	Credential CredentialConfig
}

// CredentialConfig is jobcan credential
type CredentialConfig struct {
	ClientID string
	LoginID  string
	Password string
}

func main() {
	app := cli.NewApp()
	app.Name = "jobcan"
	app.Usage = "attendance operation command for jobcan"
	app.Version = "0.2.1"
	app.Commands = []cli.Command{
		{
			Name:  "init",
			Usage: "initialize to jobcan account",
			Action: func(c *cli.Context) error {

				var config Config
				var credentialConfig CredentialConfig
				credentialConfig.ClientID = prompter.Prompt("Enter your client ID", "")
				credentialConfig.LoginID = prompter.Prompt("Enter your login ID", "")
				credentialConfig.Password = prompter.Prompt("Enter your password", "")
				config.Credential = credentialConfig

				var buffer bytes.Buffer
				encoder := toml.NewEncoder(&buffer)
				_ = encoder.Encode(config)

				ioutil.WriteFile(configPath(), []byte(buffer.String()), os.ModePerm)
				return nil
			},
		},
		{
			Name:  "start",
			Usage: "I will start a job.",
			Action: func(c *cli.Context) error {
				err := execAttendance("work_start")
				if err != nil {
					return cli.NewExitError(err, 1)
				}
				return nil
			},
		},
		{
			Name:  "end",
			Usage: "Today's work is over!",
			Action: func(c *cli.Context) error {
				err := execAttendance("work_end")
				if err != nil {
					return cli.NewExitError(err, 1)
				}
				return nil
			},
		},
		{
			Name:  "list",
			Usage: "Get your attendance list",
			Action: func(c *cli.Context) error {
				err := execGetAttendance()
				if err != nil {
					return cli.NewExitError(err, 1)
				}
				return nil
			},
		},
	}

	app.Run(os.Args)

}

func trimMetaChars(str string) string {
	r := strings.NewReplacer("\n", "", "\t", "", " ", "")
	return r.Replace(str)
}

func configPath() string {
	// only OSX
	usr, _ := user.Current()
	return strings.Replace("~/.jobcan", "~", usr.HomeDir, 1)
}

func execAttendance(mode string) error {
	var config Config

	_, err := toml.DecodeFile(configPath(), &config)
	if err != nil {
		return fmt.Errorf("Config file is broken ;; please try `jobcan init`.")
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatal(err)
	}

	client := &http.Client{Jar: jar}
	login(client, config.Credential.ClientID, config.Credential.LoginID, config.Credential.Password)
	token, groupID := fetchTokenAndGroup(client)
	pushDakoku(client, mode, token, groupID)

	fmt.Println("done!")
	fmt.Println("see https://ssl.jobcan.jp/employee/")

	return nil
}

func execGetAttendance() error {
	var config Config

	_, err := toml.DecodeFile(configPath(), &config)
	if err != nil {
		return fmt.Errorf("Config file is broken ;; please try `jobcan init`.")
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatal(err)
	}

	client := &http.Client{Jar: jar}
	login(client, config.Credential.ClientID, config.Credential.LoginID, config.Credential.Password)

	res, err := client.Get("https://ssl.jobcan.jp/employee/attendance")
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	table := tablewriter.NewWriter(os.Stdout)

	doc, _ := goquery.NewDocumentFromReader(res.Body)

	head := []string{}
	doc.Find(".note tbody tr:first-child th").Each(func(i int, s *goquery.Selection) {
		head = append(head, trimMetaChars(s.Text()))
	})
	table.SetHeader(head)

	len := doc.Find(".note tbody tr").Size() - 1
	doc.Find(".note tbody tr").Each(func(i int, s *goquery.Selection) {
		if i < len {
			data := []string{}
			s.Find("td,th").Each(func(i int, s *goquery.Selection) {
				data = append(data, trimMetaChars(s.Text()))
			})
			table.Append(data)
		}

	})

	foot := []string{}
	doc.Find(".note tbody tr:last-child th, .note tbody tr:last-child td").Each(func(i int, s *goquery.Selection) {
		foot = append(foot, trimMetaChars(s.Text()))
	})
	table.SetFooter(foot)

	table.Render()

	return nil
}

func login(client *http.Client, clientID, loginID, password string) {
	values := url.Values{}
	values.Add("client_login_id", clientID)
	values.Add("client_manager_login_id", loginID)
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

func pushDakoku(client *http.Client, mode string, token string, groupID string) {
	values := url.Values{}
	values.Add("is_yakin", "0")
	values.Add("adit_item", mode)
	values.Add("notice", "")
	values.Add("token", token)
	values.Add("adit_groupID", groupID)
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
	groupID, _ := doc.Find("select#adit_groupID option:first-child").Attr("value")
	return token, groupID
}
