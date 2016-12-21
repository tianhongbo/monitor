package main

import (
	//	"bytes"
	//	"encoding/json"
	//	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	//"io"
	//"io/ioutil"
	"log"
	//	"net/http"
	//"os"
	//"strconv"
	//"time"
)

const RM_EMULATOR_BASE_URI = "http://mtaas-worker.us-west-2.elasticbeanstalk.com/api/v1/emulator/"
const EC2_STATUS_RUNNING = "running"
const EC2_STATUS_DERMINATED = "derminated"

type ec2_t struct {
	Id            string `json:"_id"`
	Status        string `json:"status"`
	Region        string `json:"region"`
	ImageId       string `json:"image_id"`
	InstanceType  string `json:"instance_type"`
	InstanceId    string `json:"instance_id"`
	PublicIp      string `json:"public_ip"`
	PrivateIp     string `json:"private_ip"`
	KeyPair       string `json:"key_pair"`
	SecurityGroup string `json:"security_group"`
	Tag           string `json:"tag"`
}

func init() {
}

// ec2 constructor
func NewEc2(region string, imageId string, instanceType string, key string, sec string, tag string) *ec2_t {

	return &ec2_t{Region: region, ImageId: imageId, InstanceType: instanceType, KeyPair: key, SecurityGroup: sec, Tag: tag}

}

func (e *ec2_t) isRunning() bool {

	if e.Status == EC2_STATUS_RUNNING {
		return true
	} else {
		return false
	}
}

// populate public and private IP address of ec2 instance
func (e *ec2_t) getIpAddr() {
	svc := ec2.New(session.New(&aws.Config{Region: aws.String(e.Region)}))
	params := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			/*
				&ec2.Filter{
					Name: aws.String("instance-state-name"),
					Values: []*string{
						aws.String("running"),
						aws.String("pending"),
					},
				},
			*/
			&ec2.Filter{
				Name: aws.String("instance-id"),
				Values: []*string{
					aws.String(e.InstanceId),
				},
			},
		},
	}

	resp, err := svc.DescribeInstances(params)
	if err != nil {
		log.Println("fail to get public and private IP address of ec2 instance. error: ", err)
		return
	}

	instance := resp.Reservations[0].Instances[0]
	e.PrivateIp = *instance.PrivateIpAddress
	e.PublicIp = *instance.PublicIpAddress
	log.Println("successfully get ec2 instance's IP address. public ip: ", e.PublicIp, " private ip: ", e.PrivateIp)
	return

}

func (e *ec2_t) launch() error {

	svc := ec2.New(session.New(&aws.Config{Region: aws.String(e.Region)}))
	// Specify the details of the instance that you want to create.
	runResult, err := svc.RunInstances(&ec2.RunInstancesInput{
		// An Amazon Linux AMI ID for t2.micro instances in the us-west-2 region
		ImageId:        aws.String(e.ImageId),
		InstanceType:   aws.String(e.InstanceType),
		KeyName:        aws.String(e.KeyPair),
		SecurityGroups: []*string{aws.String(e.SecurityGroup)},
		MinCount:       aws.Int64(1),
		MaxCount:       aws.Int64(1),
	})

	if err != nil {
		log.Println("Could not create instance. error: ", err)
		return err
	}
	e.InstanceId = *runResult.Instances[0].InstanceId
	//e.PublicIp = *runResult.Instances[0].PublicIpAddress

	//e.PrivateIp = *runResult.Instances[0].PrivateIpAddress
	//log.Println("resp: ", runResult)

	// Add tags to the created instance
	_, errtag := svc.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{&e.InstanceId},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String(e.Tag),
			},
		},
	})
	if errtag != nil {
		log.Println("Could not create tags for instance", e.InstanceId, errtag)
		return errtag
	}

	log.Println("Successfully tagged instance with id: ", e.InstanceId)

	log.Println("Successfully create one instance. id: ", e.InstanceId)
	return nil
}

// terminate ec2
func (e *ec2_t) terminate() {

	svc := ec2.New(session.New(&aws.Config{Region: aws.String(e.Region)}))
	params := &ec2.TerminateInstancesInput{
		InstanceIds: []*string{ // Required
			aws.String(e.InstanceId), // Required
			// More values...
		},
	}
	_, err := svc.TerminateInstances(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		log.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	//log.Println(resp)

	// Pretty-print the response data.
	log.Println("terminate one ec2 instance. id: ", e.InstanceId)

}
