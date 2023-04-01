package main

import (
	"bytes"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/urfave/cli/v2"
)

func login(host, username, password string) (string, error) {
	form := url.Values{}
	form.Set("timeclock", fmt.Sprintf("%d", time.Now().Unix()))
	form.Set("luci_username", username)
	form.Set("luci_password", password)
	body := strings.NewReader(form.Encode())

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	u, _ := url.Parse("http://192.168.1.254/cgi-bin/luci")
	u.Host = host

	log.Printf("login to %s...\n", u.String())

	req, err := http.NewRequest(http.MethodPost, u.String(), body)
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if resp.StatusCode != http.StatusFound {
		return "", fmt.Errorf("login error: %d %s\n", resp.StatusCode, resp.Status)
	}

	cookie := resp.Header.Get("Set-Cookie")
	splitted := strings.Split(cookie, ";")
	// return sysauth=xxx
	return splitted[0], nil
}

func getCSRFHiddenToken(host, authCookie string) (string, error) {
	u, _ := url.Parse("http://192.168.1.254/cgi-bin/luci/admin/network/gcom/atcmd?detail=&iface=4g")
	u.Host = host

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Cookie", authCookie)

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("get hidden token err: %s\n", resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}
	hiddenToken := doc.Find("form input[name=\"token\"]").Get(0)
	if hiddenToken == nil {
		return "", fmt.Errorf("no csrf hidden token found")
	}
	for _, attr := range hiddenToken.Attr {
		if attr.Key == "value" {
			return attr.Val, nil
		}
	}
	return "", fmt.Errorf("no value inside hidden token")
}

func runListSmsATCommand(host string, authCookie string, hiddenToken string) (string, error) {
	body := &bytes.Buffer{}
	form := multipart.NewWriter(body)
	_ = form.WriteField("token", hiddenToken)
	_ = form.WriteField("cbi.submit", "1")
	_ = form.WriteField("cbid.atcmd.1.command", "at+cmgl=all")
	_ = form.WriteField("cbid.atcmd.1._custom", "")
	_ = form.WriteField("cbid.atcmd.1.refresh", "AT Command")
	if err := form.Close(); err != nil {
		return "", err
	}

	u, _ := url.Parse("http://192.168.1.254/cgi-bin/luci/admin/network/gcom/atcmd?detail=&iface=4g")
	u.Host = host

	req, err := http.NewRequest(http.MethodPost, u.String(), body)
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", form.FormDataContentType())
	req.Header.Add("Cookie", authCookie)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("runAtcommand err: %s", resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}

	textarea := doc.Find("form textarea").Get(0)
	if textarea == nil || textarea.FirstChild == nil {
		return "", fmt.Errorf("no at command textarea response")
	}

	return textarea.FirstChild.Data, nil
}

func main() {
	app := getCliApp()
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

type Session struct {
	AuthCookie       string
	ATCmdHiddenToken string
}

func loginHandler(host string, password string) (*Session, error) {
	authCookie, err := login(host, "admin", password)
	if err != nil {
		log.Printf("login failed")
		return nil, err
	}

	hiddenToken, err := getCSRFHiddenToken(host, authCookie)
	if err != nil {
		return nil, err
	}
	return &Session{
		AuthCookie:       authCookie,
		ATCmdHiddenToken: hiddenToken,
	}, nil
}

func listAllSmsHandler(host string, sess *Session) error {
	listSmsCmdResult, err := runListSmsATCommand(host, sess.AuthCookie, sess.ATCmdHiddenToken)
	if err != nil {
		return err
	}

	smsList, err := ParseATCommandSMSList(listSmsCmdResult)
	if err != nil {
		return err
	}

	fmt.Println("List All Received SMS")
	for idx, sms := range smsList {
		fmt.Printf("# %d - %s\n", idx+1, sms.Status)
		fmt.Printf("From: %s, %s\n", sms.Sender, sms.Date)
		fmt.Printf("Text: %s\n", sms.Content)
	}
	return nil
}

func getCliApp() *cli.App {
	flags := []cli.Flag{
		&cli.StringFlag{
			Name:        "host",
			Usage:       "IP address router for accessing Admin Panel",
			DefaultText: "192.168.1.254",
		},
		&cli.StringFlag{
			Name:     "password",
			Usage:    "Admin panel password",
			Required: true,
		},
	}

	listSmsCmd := &cli.Command{
		Name:  "list",
		Usage: "List all received SMS",
		Flags: flags,
		Action: func(c *cli.Context) error {
			host := c.String("host")
			if host == "" {
				host = "192.168.1.254"
			}
			password := c.String("password")

			sess, err := loginHandler(host, password)
			if err != nil {
				return err
			}
			return listAllSmsHandler(host, sess)
		},
	}

	return &cli.App{
		Name:  "cpe200sms",
		Usage: "Utilities for working with HSAirPo CPE200 SMS",
		Commands: []*cli.Command{
			listSmsCmd,
		},
	}
}
