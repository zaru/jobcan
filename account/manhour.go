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

	chooseDay := u.promptChooseDay(targetDayLists)
	fmt.Println(chooseDay)

	return nil
}

func (u *user) promptChooseDay(targetDayLists []string) int64 {
	targetTime := ""
	prompt := &survey.Select{
		Message: "Choose a time:",
		Options: targetDayLists,
	}
	survey.AskOne(prompt, &targetTime, nil)
	return strToUnixTime(targetTime)
}

func strToUnixTime(str string) int64 {
	r := regexp.MustCompile(`([0-9]{4}/[0-9]{2}/[0-9]{2})`)
	result := r.FindAllStringSubmatch(str, -1)
	t, _ := time.Parse("2006/01/02", result[0][1])
	return t.Unix()
}
