package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
)

type SMS struct {
	Status  string
	Sender  string
	Date    string
	Content string
}

// ParseSMSItem this format:
// info: +CMGL: 0,"REC UNREAD","002B0036003200380031003200320031003400380034003800330031",,"23/04/01,07:08:10+28"
// payload: 0047006F006F
func ParseSMSItem(info, payload string) (SMS, error) {
	record, err := csv.NewReader(strings.NewReader(info)).Read()
	if err != nil {
		return SMS{}, err
	}
	if len(record) < 4 {
		return SMS{}, fmt.Errorf("info records expect 4 segments")
	}

	s := SMS{}
	s.Status = record[1]
	s.Sender = hexToString(record[2])
	s.Date = record[4]
	s.Content = hexToString(payload)
	return s, nil
}

func hexToString(hexString string) string {
	str, err := hex.DecodeString(hexString)
	if err != nil {
		log.Printf("hexToString err: %s\n", err)
		return ""
	}
	return string(bytes.ReplaceAll(str, []byte{0}, []byte{}))
}

func ParseATCommandSMSList(atCmdResult string) ([]SMS, error) {
	var smsList []SMS
	reader := bufio.NewReader(strings.NewReader(atCmdResult))
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			break
		}
		lineStr := string(line)
		if strings.HasPrefix(lineStr, "+CMGL:") {
			payload, _, err := reader.ReadLine()
			if err != nil {
				break
			}

			sms, err := ParseSMSItem(lineStr, string(payload))
			if err != nil {
				return nil, err
			}
			smsList = append(smsList, sms)
		}
	}

	return smsList, nil
}
