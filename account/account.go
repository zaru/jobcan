package account

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/olekukonko/tablewriter"
	"github.com/zaru/jobcan/client"
	"github.com/zaru/jobcan/config"
	"gopkg.in/AlecAivazis/survey.v1"
)

type AccountType int

const (
	General = iota
	Admin
)

func New(at AccountType) Account {

	config, err := config.Read()
	if err != nil {
		log.Fatal(err)
	}

	httpClient := client.New()

	u := &user{
		clientID:   config.Credential.ClientID,
		loginID:    config.Credential.LoginID,
		password:   config.Credential.Password,
		httpClient: httpClient,
	}
	if at == Admin {
		return &admin{u}
	}

	return &admin{u}
}

type Account interface {
	Login()
	ExecAttendance(mode string)
	ExecGetAttendance() error
	ExecGetAttendanceByDay(day string) error

	promptFix() bool
	promptChooseTime(targetTimeLists map[string]string) string
	promptFixTime() string
	formatFixTimeParams(doc *goquery.Document) FixTimeParams
	sendFixTime(params FixTimeParams)
	pushDakoku(mode string, token string, groupID string)
	fetchTokenAndGroup() (string, string)
}

type user struct {
	clientID   string
	loginID    string
	password   string
	httpClient *http.Client
}

type admin struct {
	*user
}

type FixTimeParams struct {
	Token        string
	DeleteMinute string
	Time         string
	GroupId      string
	Notice       string
	Year         string
	Month        string
	Day          string
	ClientId     string
	EmployeeId   string
}

func trimMetaChars(str string) string {
	r := strings.NewReplacer("\n", "", "\t", "", " ", "")
	return r.Replace(str)
}

func (u *user) ExecAttendance(mode string) {
	token, groupID := u.fetchTokenAndGroup()
	u.pushDakoku(mode, token, groupID)

	fmt.Println("done!")
	fmt.Println("see https://ssl.jobcan.jp/employee/")
}

func (u *user) ExecGetAttendance() error {
	res, err := u.httpClient.Get("https://ssl.jobcan.jp/employee/attendance")
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

func (u *user) ExecGetAttendanceByDay(day string) error {
	r := regexp.MustCompile(`([0-9]{4})-?([0-9]{2})-?([0-9]{2})`)
	result := r.FindAllStringSubmatch(day, -1)

	res, err := u.httpClient.Get(fmt.Sprintf("https://ssl.jobcan.jp/employee/adit/modify?year=%s&month=%s&day=%s",
		result[0][1], result[0][2], result[0][3]))
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	table := tablewriter.NewWriter(os.Stdout)

	// display column size
	maxColumn := 4
	// action column index number
	actionColumnIndex := 5

	doc, _ := goquery.NewDocumentFromReader(res.Body)

	head := []string{}
	doc.Find(".note tbody tr:first-child th").Each(func(i int, s *goquery.Selection) {
		if i < maxColumn {
			head = append(head, trimMetaChars(s.Text()))
		}
	})
	table.SetHeader(head)

	targetTimeLists := map[string]string{"cancel": "0"}
	doc.Find(".note tbody tr").Each(func(i int, s *goquery.Selection) {
		if i > 0 {
			data := []string{}
			s.Find("td").Each(func(i int, s *goquery.Selection) {
				if i < maxColumn {
					data = append(data, trimMetaChars(s.Text()))
				} else if i == actionColumnIndex {
					data, err := s.Find("a.btn-info").Attr("onclick")
					if err != false {
						r := regexp.MustCompile(`intoModifyMode\(([0-9]+), '([0-9]{2}:[0-9]{2})'`)
						result := r.FindAllStringSubmatch(data, -1)
						targetTimeLists[result[0][2]] = result[0][1]
					}
				}
			})
			table.Append(data)
		}

	})

	table.Render()

	fixFlag := u.promptFix()
	if fixFlag == false {
		return nil
	}

	targetTime := u.promptChooseTime(targetTimeLists)
	if targetTime == "cancel" {
		return nil
	}

	fixTime := u.promptFixTime()

	params := u.formatFixTimeParams(doc)
	params.DeleteMinute = targetTimeLists[targetTime]
	params.Time = fixTime
	u.sendFixTime(params)

	return nil
}

func (u *user) promptFix() bool {
	fixFlag := false
	prompt := &survey.Confirm{
		Message: "Fix it?",
	}
	survey.AskOne(prompt, &fixFlag, nil)
	return fixFlag
}

func (u *user) promptChooseTime(targetTimeLists map[string]string) string {
	targetTime := ""
	keys := make([]string, 0, len(targetTimeLists))
	for k := range targetTimeLists {
		keys = append(keys, k)
	}
	prompt := &survey.Select{
		Message: "Choose a time:",
		Options: keys,
	}
	survey.AskOne(prompt, &targetTime, nil)
	return targetTime
}

func (u *user) promptFixTime() string {
	time := ""
	prompt := &survey.Input{
		Message: "Input a time (HHMM)",
	}
	survey.AskOne(prompt, &time, nil)
	return time
}

func (u *user) formatFixTimeParams(doc *goquery.Document) FixTimeParams {

	token, _ := doc.Find("input[name=token]").Attr("value")
	year, _ := doc.Find("input[name=year]").Attr("value")
	month, _ := doc.Find("input[name=month]").Attr("value")
	day, _ := doc.Find("input[name=day]").Attr("value")
	clientId, _ := doc.Find("input[name=client_id]").Attr("value")
	employeeId, _ := doc.Find("input[name=employee_id]").Attr("value")
	groupId, _ := doc.Find("select[name=group_id] option:first-child").Attr("value")
	return FixTimeParams{
		Token:      token,
		GroupId:    groupId,
		Year:       year,
		Month:      month,
		Day:        day,
		ClientId:   clientId,
		EmployeeId: employeeId,
	}
}

func (u *user) sendFixTime(params FixTimeParams) {
	values := url.Values{}
	values.Add("token", params.Token)
	values.Add("delete_minutes", params.DeleteMinute)
	values.Add("time", params.Time)
	values.Add("group_id", params.GroupId)
	values.Add("notice", "fix")
	values.Add("year", params.Year)
	values.Add("month", params.Month)
	values.Add("day", params.Day)
	values.Add("client_id", params.ClientId)
	values.Add("employee_id", params.EmployeeId)
	res, err := u.httpClient.PostForm("https://ssl.jobcan.jp/employee/adit/insert/", values)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatal("Post error StatusCode=" + string(res.StatusCode))
	}
}

func (u *user) pushDakoku(mode string, token string, groupID string) {
	values := url.Values{}
	values.Add("is_yakin", "0")
	values.Add("adit_item", mode)
	values.Add("notice", "")
	values.Add("token", token)
	values.Add("adit_groupID", groupID)
	res, err := u.httpClient.PostForm("https://ssl.jobcan.jp/employee/index/adit", values)
	if err != nil {
		log.Fatal(err)
	}

	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatal("Post error StatusCode=" + string(res.StatusCode))
		return
	}
}

func (u *user) fetchTokenAndGroup() (string, string) {
	res, err := u.httpClient.Get("https://ssl.jobcan.jp/employee")
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	doc, _ := goquery.NewDocumentFromReader(res.Body)
	token, _ := doc.Find("input[name='token']").Attr("value")
	groupID, _ := doc.Find("select#adit_groupID option:first-child").Attr("value")
	return token, groupID
}
