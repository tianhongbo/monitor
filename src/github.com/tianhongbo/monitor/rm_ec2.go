package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	//"io"
	//"io/ioutil"
	"log"
	"net/http"
	//"os"
	//"strconv"
)

const RM_EMULATOR_BASE_URI = "http://mtaas-worker.us-west-2.elasticbeanstalk.com/api/v1/emulator/"
const EMULATOR_KEY_PAIR = "scott-j"
const EMULATOR_SECURITY_GROUP = "mtaas-emulator"
const EC2_STATUS_RUNNING = "running"
const EC2_STATUS_DERMINATED = "derminated"

type ec2_t struct {
	Id           string `json:"_id"`
	Status       string `json:"status"`
	Region       string `json:"region"`
	ImageId      string `json:"image_id"`
	InstanceType string `json:"instance_type"`
	InstanceId   string `json:"instance_id"`
	PublicIp     string `json:"public_ip"`
	PrivateIp    string `json:"private_ip"`
}

func init() {
}

// ec2 constructor
func NewEc2(region string, imageId string, instanceType string) *ec2_t {

	return &ec2_t{Region: region, ImageId: imageId, InstanceType: instanceType}

}

func (e *ec2_t) isRunning() bool {

	if e.Status == EC2_STATUS_RUNNING {
		return true
	} else {
		return false
	}
}

func (e *ec2_t) launch() error {

	svc := ec2.New(session.New(&aws.Config{Region: aws.String(e.Region)}))
	// Specify the details of the instance that you want to create.
	runResult, err := svc.RunInstances(&ec2.RunInstancesInput{
		// An Amazon Linux AMI ID for t2.micro instances in the us-west-2 region
		ImageId:        aws.String(e.ImageId),
		InstanceType:   aws.String(e.InstanceType),
		KeyName:        aws.String(EMULATOR_KEY_PAIR),
		SecurityGroups: []*string{aws.String(EMULATOR_SECURITY_GROUP)},
		MinCount:       aws.Int64(1),
		MaxCount:       aws.Int64(1),
	})

	if err != nil {
		log.Println("Could not create instance", err)
		return err
	}
	e.InstanceId = *runResult.Instances[0].InstanceId
	//e.PublicIp = *runResult.Instances[0].PublicIpAddress

	//e.PrivateIp = *runResult.Instances[0].PrivateIpAddress
	log.Println("resp: ", runResult)
	log.Println("Successfully create  one instance. id: ", e.InstanceId,
		"public ip: ", e.PublicIp,
		" private ip: ", e.PrivateIp)

	// Add tags to the created instance
	_, errtag := svc.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{&e.InstanceId},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String("emulator-host"),
			},
		},
	})
	if errtag != nil {
		log.Println("Could not create tags for instance", e.InstanceId, errtag)
		return errtag
	}

	log.Println("Successfully tagged instance. public")

	// get public and private ip
	// of these two states. This is generally what we want
	params := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name: aws.String("instance-state-name"),
				Values: []*string{
					aws.String("running"),
					aws.String("pending"),
				},
			},
			&ec2.Filter{
				Name: aws.String("instance-id"),
				Values: []*string{
					aws.String(e.InstanceId),
				},
			},
		},
	}

	// TODO: Actually care if we can't connect to a host
	resp, _ := svc.DescribeInstances(params)
	// if err != nil {
	//      panic(err)
	// }

	// Loop through the instances. They don't always have a name-tag so set it
	// to None if we can't find anything.
	instance := resp.Reservations[0].Instances[0]
	e.PrivateIp = *instance.PrivateIpAddress
	//e.PublicIp = *instance.PublicIpAddress
log.Println("instance: ", resp)	
log.Println("e :", e)
	return nil
}

// handle one message
func (e *ec2_t) doOneMsg() error {
	emu, err := getOneEmulator()
	if err != nil {
		// do not delete the message from sqs queue
		log.Println("fail to allocate a emulator.")
		return errors.New("fail to allocate a emlator.")
	}
	log.Println(e.Id, emu.Name)

	// reuse req to assemble rsp
	e.Status = "occupied"

	// code to json
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(e)

	// set url
	url := RSP_EMULATOR_URI + e.Id

	// send PUT request
	client := &http.Client{}
	request, err := http.NewRequest("PUT", url, b)
	request.Header.Set("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		log.Println(err)
		return errors.New("fail to send PUT request.")
	}
	log.Println("send emulator req to: ", url)
	log.Println("receive emulator rsp: ", response)
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
	resp, err := svc.TerminateInstances(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		log.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	log.Println(resp)

	// Pretty-print the response data.
	log.Println("delete one message.")

}
