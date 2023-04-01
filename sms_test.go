package main

import (
	"os"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestName(t *testing.T) {
	f, err := os.Open("sample_at_cmd_resp.html")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		t.Fatal(err)
	}

	textarea := doc.Find("form textarea").Get(0)
	t.Log(textarea.FirstChild.Data)
}

func Test_ParseATCommandSMSList(t *testing.T) {
	data := `at+cmgl=all

+CMGL: 0,"REC UNREAD","002B0036003200380031003200320031003400380034003800330031",,"23/04/01,07:08:10+28"
0047006F006F

OK
`

	smsList, err := ParseATCommandSMSList(data)
	if err != nil {
		t.Fatal(err)
	}

	for idx, sms := range smsList {
		t.Logf("%d. %+v\n", idx+1, sms)
	}
}
