package main

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go/service/ses"
)

// informOwner sends the cluster owner a mail,
// which can be used to give them heads-up about
// an upcoming tear down, for example.
func informOwner(tomail, subject, body string) error {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return err
	}
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
		Source: aws.String("hausenbl@amazon.com"),
	})
	_, err := req.Send(context.Background())
	if err != nil {
		return err
	}
	return nil
}
