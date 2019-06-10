package main

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ses"
)

// informOwner sends the cluster owner a mail,
// which can be used to give them heads-up about
// an upcoming tear down, for example.
func informOwner(tomail, subject, body string) error {
	// get the source email address from env provided by user at install time:
	srcmailaddress := os.Getenv("NOTIFICATION_EMAIL_ADDRESS")
	// if no source email address provided this is a NOP:
	if srcmailaddress == "" {
		return nil
	}
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return err
	}
	// have to pick a region where SES is available:
	cfg.Region = "eu-west-1"
	svc := ses.New(cfg)
	req := svc.SendEmailRequest(&ses.SendEmailInput{
		Destination: &ses.Destination{ToAddresses: []string{tomail}},
		Message: &ses.Message{
			Body: &ses.Body{
				Text: &ses.Content{
					Charset: aws.String("UTF-8"),
					Data:    aws.String(body),
				},
			},
			Subject: &ses.Content{
				Charset: aws.String("UTF-8"),
				Data:    aws.String(subject),
			},
		},
		Source: &srcmailaddress,
	})
	_, err = req.Send(context.Background())
	if err != nil {
		return err
	}
	return nil
}
