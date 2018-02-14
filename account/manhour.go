package account

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/olekukonko/tablewriter"
	"gopkg.in/AlecAivazis/survey.v1"
)

func (u *user) ExecGetManHour(day string) error {
	r := regexp.MustCompile(`([0-9]{4})-?([0-9]{2})`)
	result := r.FindAllStringSubmatch(day, -1)

	year := strconv.Itoa(time.Now().Year())

	values := url.Values{}
	if len(result) > 0 {
		year = result[0][1]
		values.Add("year", result[0][1])
		values.Add("month", result[0][2])
	}
	res, err := u.httpClient.PostForm("https://ssl.jobcan.jp/employee/man-hour-manage", values)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	table := tablewriter.NewWriter(os.Stdout)

	doc, _ := goquery.NewDocumentFromReader(res.Body)

	// display column size
	maxColumn := 3

	head := []string{}
	doc.Find(".man-hour-table tbody tr:first-child th").Each(func(i int, s *goquery.Selection) {
		if i < maxColumn {
			head = append(head, trimMetaChars(s.Text()))
		}
	})
	table.SetHeader(head)

	targetDayLists := []string{"cancel"}
	doc.Find(".man-hour-table tbody tr").Each(func(i int, s *goquery.Selection) {
		if i > 0 {
			data := []string{}
			s.Find("td,th").Each(func(i int, s *goquery.Selection) {
				if i < maxColumn {
					data = append(data, year+"/"+trimMetaChars(s.Text()))
				}
				if i == 0 {
					targetDayLists = append(targetDayLists, year+"/"+trimMetaChars(s.Text()))
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

	chooseDay := promptChooseDay(targetDayLists)
	fmt.Println(chooseDay)

	doc = u.fetchManHourFormDoc(chooseDay)
	token, projects := fetchManHourTokenAndProjects(doc)
	project := promptChooseProject(projects)
	tasks := fetchManHourTasks(doc, projects[project])
	fmt.Println(token)
	fmt.Println(project)
	fmt.Println(tasks)

	return nil
}

func promptChooseDay(targetDayLists []string) string {
	targetTime := ""
	prompt := &survey.Select{
		Message: "Choose a time:",
		Options: targetDayLists,
	}
	survey.AskOne(prompt, &targetTime, nil)
	return strconv.FormatInt(strToUnixTime(targetTime), 10)
}

func promptChooseProject(projects map[string]string) string {
	var targetProject string
	keys := make([]string, 0, len(projects))
	for k := range projects {
		keys = append(keys, k)
	}
	prompt := &survey.Select{
		Message: "Choose a project:",
		Options: keys,
	}
	survey.AskOne(prompt, &targetProject, nil)
	return targetProject
}

func strToUnixTime(str string) int64 {
	r := regexp.MustCompile(`([0-9]{4}/[0-9]{2}/[0-9]{2})`)
	result := r.FindAllStringSubmatch(str, -1)
	t, _ := time.Parse("2006/01/02", result[0][1])
	return t.Unix()
}

type ManHourForm struct {
	Error bool   `json:"error"`
	Html  string `json:"html"`
}

func (u *user) fetchManHourFormDoc(ts string) *goquery.Document {
	res, err := u.httpClient.Get("https://ssl.jobcan.jp/employee/man-hour-manage/get-man-hour-data-for-edit/unix_time/" + ts)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	doc, _ := goquery.NewDocumentFromReader(res.Body)
	return doc
}

func fetchManHourTokenAndProjects(doc *goquery.Document) (string, map[string]string) {
	token, _ := doc.Find("").Attr("value")

	projects := map[string]string{}
	doc.Find("select[name='projects[]'] option").Each(func(i int, s *goquery.Selection) {
		val, _ := s.Attr("value")
		projects[s.Text()] = val
	})
	return token, projects
}

func fetchManHourTasks(doc *goquery.Document, projectID string) map[string]string {
	projects := map[string]string{}
	doc.Find("#task-list-" + projectID + " option").Each(func(i int, s *goquery.Selection) {
		val, _ := s.Attr("value")
		projects[s.Text()] = val
	})
	return projects
}

func (u *user) pushManHour(token string) {
	values := url.Values{}
	values.Add("token", token)
	values.Add("template", "")
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
