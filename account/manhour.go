package account

import (
	"log"
	"net/url"
	"os"
	"regexp"

	"github.com/PuerkitoBio/goquery"
	"github.com/olekukonko/tablewriter"
)

func (u *user) ExecGetManHour(day string) error {
	r := regexp.MustCompile(`([0-9]{4})-?([0-9]{2})`)
	result := r.FindAllStringSubmatch(day, -1)

	values := url.Values{}
	if len(result) > 0 {
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

	doc.Find(".man-hour-table tbody tr").Each(func(i int, s *goquery.Selection) {
		if i > 0 {
			data := []string{}
			s.Find("td,th").Each(func(i int, s *goquery.Selection) {
				if i < maxColumn {
					data = append(data, trimMetaChars(s.Text()))
				}
			})
			table.Append(data)
		}

	})

	table.Render()

	return nil
}
