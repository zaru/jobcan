package account

import (
	"log"
	"net/url"
	"regexp"

	"github.com/PuerkitoBio/goquery"
)

func (u *admin) Login() {
	values := url.Values{}
	values.Add("client_login_id", u.clientID)
	values.Add("client_manager_login_id", u.loginID)
	values.Add("client_login_password", u.password)
	values.Add("login_type", "2")
	values.Add("url", "https://ssl.jobcan.jp/client/")
	res, err := u.httpClient.PostForm("https://ssl.jobcan.jp/login/client", values)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatal("Login error StatusCode=" + string(res.StatusCode))
	}
	u.employeeLogin()
}

func (u *admin) employeeLogin() {
	code := u.fetchEmployeeCode()
	res, err := u.httpClient.Get("https://ssl.jobcan.jp/login/pc-employee/try?code=" + code)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatal("Login error StatusCode=" + string(res.StatusCode))
	}
}

func (u *admin) fetchEmployeeCode() string {
	res, err := u.httpClient.Get("https://ssl.jobcan.jp/client")
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
