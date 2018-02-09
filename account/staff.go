package account

import (
	"log"
	"net/url"
)

func (u *staff) Login() {
	values := url.Values{}
	values.Add("client_id", u.clientID)
	values.Add("email", u.loginID)
	values.Add("password", u.password)
	values.Add("login_type", "1")
	values.Add("url", "/employee")
	res, err := u.httpClient.PostForm("https://ssl.jobcan.jp/login/pc-employee", values)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatal("Login error StatusCode=" + string(res.StatusCode))
	}
}
