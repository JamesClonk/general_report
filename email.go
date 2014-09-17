package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net/smtp"
	"strings"
)

func sendEmails(query *Query) {
	var files []string
	if query.Output != nil {
		for _, csv := range query.Output.Csv {
			if csv.Mail {
				files = append(files, csv.Value)
			}
		}
		for _, xls := range query.Output.Xls {
			if xls.Mail {
				files = append(files, xls.Value)
			}
		}
	}
	if len(files) > 0 || query.Email.SendEmptyReport {
		sendEmail(query.Email.From, query.Email.To, query.Email.Cc, query.Email.Subject, query.Email.Body, files)
	}
}

func sendEmail(from string, to, cc []string, subject, body string, filenames []string) {
	var msg bytes.Buffer
	var boundary = "__GENERAL_REPORT__"

	if from == "" {
		from = "default.from@localhost.localdomain" // TODO: change this value here!
	}
	if len(to) == 0 {
		to = []string{"default.to@localhost.localdomain"} // TODO: change this value here!
	}
	if subject == "" {
		subject = "General Report - " + TIMESTAMP
	}
	if body == "" {
		body = "Hello,\r\n\r\nHere's your requested report!\r\n\r\nRegards\r\nGeneral Report"
	}

	// header
	msg.WriteString(fmt.Sprintf("From: General Report <%s>\r\n", from))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(to, ",")))
	if len(cc) > 0 {
		msg.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(cc, ",")))
	}
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString(fmt.Sprintf("MIME-Version: 1.0\r\n"))
	if len(filenames) > 0 {
		msg.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=%s\r\n--%s\r\n", boundary, boundary))
	}

	// body
	msg.WriteString(fmt.Sprintf("Content-Type: text/plain; charset=utf-8\r\n\r\n%s\r\n--%s", body, boundary))

	// attachments/files
	for _, filename := range filenames {
		content, err := ioutil.ReadFile(filename)
		if err != nil {
			log.Println(err)
			EXITCODE = 21
		}
		encoded := base64.StdEncoding.EncodeToString(content)

		msg.WriteString(fmt.Sprintf("\r\nContent-Type: application/octet-stream; name=\"%s\"\r\nContent-Transfer-Encoding:base64\r\n", filename))
		msg.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n\r\n%s\r\n--%s", filename, encoded, boundary))
	}
	msg.WriteString("--")

	// send email
	err := smtp.SendMail("localhost:25", nil, from, to, msg.Bytes())
	if err != nil {
		log.Println("could not send email")
		log.Println(err)
		EXITCODE = 20
	}
}
