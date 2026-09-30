package awshelper

import (
	"bytes"
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/davecgh/go-spew/spew"
	"github.com/reverent911/kds-golang-lib/pkg/generic/helper"
)

const SQSEmail = "https://sqs.eu-west-1.amazonaws.com/965924223854/email_queue.fifo"
const COMSQueue = "https://sqs.eu-west-1.amazonaws.com/965924223854/communications"
const ContactlessQueue = "https://sqs.eu-west-1.amazonaws.com/965924223854/contactless"
const YLAQueue = "https://sqs.eu-west-1.amazonaws.com/965924223854/yourloyaltyapp"

func KDSUploadReport(filename string, bucketname string, key string) error {
	awsConfig := &aws.Config{
		Credentials: credentials.NewStaticCredentials(os.Getenv("AWS_ACCESS_KEY_ID_REPORTS"), os.Getenv("AWS_SECRET_ACCESS_KEY_REPORTS"), ""),
		Region:      aws.String("eu-west-1"),
	}
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            *awsConfig,
	}))
	s3command := s3.New(sess)
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	fileInfo, _ := file.Stat()
	var size int64 = fileInfo.Size()
	buffer := make([]byte, size)
	file.Read(buffer)
	spew.Dump(size)

	a, err := s3command.PutObject(&s3.PutObjectInput{
		Bucket:               aws.String(bucketname),
		Key:                  aws.String(key),
		ACL:                  aws.String("public-read"),
		Body:                 bytes.NewReader(buffer),
		ContentLength:        aws.Int64(size),
		ContentType:          aws.String(http.DetectContentType(buffer)),
		ContentDisposition:   aws.String("attachment"),
		ServerSideEncryption: aws.String("AES256"),
	})
	spew.Dump(a)
	return err
}

func KDSTerminalUploadFile(filename string, bucketname string, key string) error {
	awsConfig := &aws.Config{
		Credentials: credentials.NewStaticCredentials(os.Getenv("AWS_ACCESS_KEY_ID_TERMINAL"), os.Getenv("AWS_SECRET_ACCESS_KEY_TERMINAL"), ""),
		Region:      aws.String("eu-west-1"),
	}
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            *awsConfig,
	}))
	s3command := s3.New(sess)
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	fileInfo, _ := file.Stat()
	var size int64 = fileInfo.Size()
	buffer := make([]byte, size)
	file.Read(buffer)

	_, err = s3command.PutObject(&s3.PutObjectInput{
		Bucket:               aws.String(bucketname),
		Key:                  aws.String(key),
		ACL:                  aws.String("private"),
		Body:                 bytes.NewReader(buffer),
		ContentLength:        aws.Int64(size),
		ContentType:          aws.String(http.DetectContentType(buffer)),
		ContentDisposition:   aws.String("attachment"),
		ServerSideEncryption: aws.String("AES256"),
	})
	return err
}

func YLASESSendEmail(recip string, subject string, body string) {
	CharSet := "UTF-8"
	awsConfig := &aws.Config{
		Credentials: credentials.NewStaticCredentials(os.Getenv("AWS_ACCESS_KEY_ID_MESSAGING"), os.Getenv("AWS_SECRET_ACCESS_KEY_MESSAGING"), ""),
		Region:      aws.String("eu-west-1"),
	}
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            *awsConfig,
	}))
	svc := ses.New(sess)
	sender := "yourloyaltyapp@tabit.co.za"
	input := &ses.SendEmailInput{
		Destination: &ses.Destination{
			CcAddresses: []*string{},
			ToAddresses: []*string{
				aws.String(recip),
			},
		},
		Message: &ses.Message{
			Body: &ses.Body{
				Html: &ses.Content{
					Charset: aws.String(CharSet),
					Data:    aws.String(body),
				},
				Text: &ses.Content{
					Charset: aws.String(CharSet),
					Data:    aws.String(body),
				},
			},
			Subject: &ses.Content{
				Charset: aws.String(CharSet),
				Data:    aws.String(subject),
			},
		},
		Source: aws.String(sender),
		// Uncomment to use a configuration set
		//ConfigurationSetName: aws.String(ConfigurationSet),
	}
	result, err := svc.SendEmail(input)
	spew.Dump(result)
	spew.Dump(err)
}

// SendSqsEmail ... Send Email Message to SQS Queue
func SendSqsEmail(message string, orderid int) {
	awsConfig := &aws.Config{
		Credentials: credentials.NewStaticCredentials(os.Getenv("AWS_ACCESS_KEY_ID_MESSAGING"), os.Getenv("AWS_SECRET_ACCESS_KEY_MESSAGING"), ""),
		Region:      aws.String("eu-west-1"),
	}
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            *awsConfig,
	}))
	svc := sqs.New(sess)
	res, err := svc.SendMessage(&sqs.SendMessageInput{
		DelaySeconds: aws.Int64(10),
		MessageAttributes: map[string]*sqs.MessageAttributeValue{
			"Type": &sqs.MessageAttributeValue{
				DataType:    aws.String("String"),
				StringValue: aws.String("EMAIL"),
			},
			"ID": &sqs.MessageAttributeValue{
				DataType:    aws.String("Number"),
				StringValue: aws.String(helper.IntToString(orderid)),
			},
		},
		MessageBody: aws.String(message),
		QueueUrl:    aws.String(COMSQueue),
	})
	if err != nil {
		spew.Dump(err)
	}
	spew.Dump(res)
}

// SendCONTACTLESSMenuSyncMessage ... Send CONTACTLESS Menu Sync
func SendCONTACTLESSMenuSyncMessage(ikentooID string, ikentooBusinessID string, synctype string) {
	awsConfig := &aws.Config{
		Credentials: credentials.NewStaticCredentials(os.Getenv("AWS_ACCESS_KEY_ID_MESSAGING"), os.Getenv("AWS_SECRET_ACCESS_KEY_MESSAGING"), ""),
		Region:      aws.String("eu-west-1"),
	}
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            *awsConfig,
	}))
	svc := sqs.New(sess)
	res, err := svc.SendMessage(&sqs.SendMessageInput{
		DelaySeconds: aws.Int64(10),
		MessageAttributes: map[string]*sqs.MessageAttributeValue{
			"ACTION": &sqs.MessageAttributeValue{
				DataType:    aws.String("String"),
				StringValue: aws.String("MENUSYNC"),
			},
			"IKENTOOID": &sqs.MessageAttributeValue{
				DataType:    aws.String("String"),
				StringValue: aws.String(ikentooID),
			},
			"IKENTOOBUSINESSID": &sqs.MessageAttributeValue{
				DataType:    aws.String("String"),
				StringValue: aws.String(ikentooBusinessID),
			},
			"SYNCTYPE": &sqs.MessageAttributeValue{
				DataType:    aws.String("String"),
				StringValue: aws.String(synctype),
			},
		},
		MessageBody: aws.String("OK"),
		QueueUrl:    aws.String(ContactlessQueue),
	})
	if err != nil {
		spew.Dump(err)
	}
	spew.Dump(res)
}

// SendYLAMenuSyncMessage ... Send YLA Menu Sync
func SendYLAMenuSyncMessage(ikentooID string, ikentooBusinessID string, synctype string) {
	awsConfig := &aws.Config{
		Credentials: credentials.NewStaticCredentials(os.Getenv("AWS_ACCESS_KEY_ID_MESSAGING"), os.Getenv("AWS_SECRET_ACCESS_KEY_MESSAGING"), ""),
		Region:      aws.String("eu-west-1"),
	}
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            *awsConfig,
	}))
	svc := sqs.New(sess)
	res, err := svc.SendMessage(&sqs.SendMessageInput{
		DelaySeconds: aws.Int64(10),
		MessageAttributes: map[string]*sqs.MessageAttributeValue{
			"ACTION": &sqs.MessageAttributeValue{
				DataType:    aws.String("String"),
				StringValue: aws.String("MENUSYNC"),
			},
			"IKENTOOID": &sqs.MessageAttributeValue{
				DataType:    aws.String("String"),
				StringValue: aws.String(ikentooID),
			},
			"IKENTOOBUSINESSID": &sqs.MessageAttributeValue{
				DataType:    aws.String("String"),
				StringValue: aws.String(ikentooBusinessID),
			},
			"SYNCTYPE": &sqs.MessageAttributeValue{
				DataType:    aws.String("String"),
				StringValue: aws.String(synctype),
			},
		},
		MessageBody: aws.String("OK"),
		QueueUrl:    aws.String(YLAQueue),
	})
	if err != nil {
		spew.Dump(err)
	}
	spew.Dump(res)
}

// SQSOrderCallback ... Order Callback to SQS Queue
func SQSOrderCallback(message string, orderid int) {
	awsConfig := &aws.Config{
		Credentials: credentials.NewStaticCredentials(os.Getenv("AWS_ACCESS_KEY_ID_MESSAGING"), os.Getenv("AWS_SECRET_ACCESS_KEY_MESSAGING"), ""),
		Region:      aws.String("eu-west-1"),
	}
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            *awsConfig,
	}))
	svc := sqs.New(sess)
	_, err := svc.SendMessage(&sqs.SendMessageInput{
		DelaySeconds: aws.Int64(10),
		MessageAttributes: map[string]*sqs.MessageAttributeValue{
			"Type": &sqs.MessageAttributeValue{
				DataType:    aws.String("String"),
				StringValue: aws.String("CALLBACK"),
			},
			"ID": &sqs.MessageAttributeValue{
				DataType:    aws.String("Number"),
				StringValue: aws.String(helper.IntToString(orderid)),
			},
		},
		MessageBody: aws.String(message),
		QueueUrl:    aws.String(COMSQueue),
	})
	if err != nil {
		//errors.YLASentryCapture(err)
		fmt.Println(err)
	}
}
