package main

import (
	//	"bytes"
	//	"encoding/json"
	//	"errors"
	//	"github.com/aws/aws-sdk-go/aws"
	//	"github.com/aws/aws-sdk-go/service/sqs"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	//	"strconv"
	"time"
)

//resource management
const RM_EMULATORS_URI = "http://mtaas-worker.us-west-2.elasticbeanstalk.com/api/v1/emulator"
const RM_EMULATOR_HOSTS_URI = "http://mtaas-worker.us-west-2.elasticbeanstalk.com/api/v1/emulator_host"

const RM_HUB_HOSTS_URI = "http://mtaas-worker.us-west-2.elasticbeanstalk.com/api/v1/hub"

const RM_STATUS_RUNNING = "running"
const RM_STATUS_AVAILABLE = "available"
const RM_STATUS_TERMINATED = "terminated"
const RM_STATUS_TERMINATING = "terminating"

const MAX_NUM_EMULATORS_PER_HOST = 10

const RSP_EMULATOR_URI = "http://mtaas-worker.us-west-2.elasticbeanstalk.com/api/v1/emulator/"

const AWS_CURRENT_REGION = "us-west-2"
const EMULATOR_EC2_TYPE = "t2.medium"
const EMULATOR_3_IMAGE_ID = "ami-574cf937"
const EMULATOR_10_IMAGE_ID = "ami-cf4df8af"
const EMULATOR_IMAGE_ID = EMULATOR_10_IMAGE_ID

const EMULATOR_KEY_PAIR = "scott-j"
const EMULATOR_SECURITY_GROUP = "mtaas-emulator"
const EUMLATOR_TAG = "emulator-host"

const HUB_KEY_PAIR = "aws-sjsu"
const HUB_SECURITY_GROUP = "MTaaS-Test"
const HUB_TAG = "hub-host"

const HUB_IMAGE_ID = "ami-4958ec29"
const HUB_EC2_TYPE = "t2.small"

const AWS_PUBLIC_IP_URI = "http://instance-data/latest/meta-data/public-ipv4"
const AWS_LOCAL_IP_URI = "http://instance-data/latest/meta-data/local-ipv4"
const AWS_AVAILABILITY_ZONE_URI = "http://instance-data/latest/meta-data/placement/availability-zone"

const SYS_LOG_FILE = "/home/ubuntu/monitor/log/sys.log"

type LocalEnv struct {
	PublicIp         string
	LocalIp          string
	AvailabilityZone string
}

var localEnv LocalEnv

func init() {
	//initialize log at the first place
	initLog()

	log.Println("system restart...")

	//initialize local env
	localEnv.PublicIp = getVmMetaData(AWS_PUBLIC_IP_URI)
	localEnv.LocalIp = getVmMetaData(AWS_LOCAL_IP_URI)
	localEnv.AvailabilityZone = getVmMetaData(AWS_AVAILABILITY_ZONE_URI)

	log.Println("local env initialization is done.")
	log.Println("pulic ip: ", localEnv.PublicIp)
	log.Println("local ip: ", localEnv.LocalIp)
	log.Println("availability zone: ", localEnv.AvailabilityZone)
}

func initLog() {
	file, err := os.OpenFile(SYS_LOG_FILE, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("fail to open log file with error:  ", err)
	}

	multi := io.MultiWriter(file, os.Stdout)
	log.SetOutput(multi)
	log.Println("log initialization is done.")
}

func getVmMetaData(uri string) string {
	//get public ip
	resp, err := http.Get(uri)
	if err != nil {
		// handle error
		log.Println("fail to get public IP.")
		return ""
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("fail to decode public ip.")
		return ""
	}
	return string(body)
}

func main() {

	m_us_west_2 := NewEmulatorMonitor("us-west-2", 70,
		RM_EMULATORS_URI,
		RM_EMULATOR_HOSTS_URI,
		"ami-cf4df8af")
	m_us_east_1 := NewEmulatorMonitor("us-east-1", 5,
		RM_EMULATORS_URI,
		RM_EMULATOR_HOSTS_URI,
		"ami-f31306e4")
	m_eu_west_1 := NewEmulatorMonitor("eu-west-1", 5,
		RM_EMULATORS_URI,
		RM_EMULATOR_HOSTS_URI,
		"ami-66240415")
	m_ap_southeast_1 := NewEmulatorMonitor("ap-southeast-1", 5,
		RM_EMULATORS_URI,
		RM_EMULATOR_HOSTS_URI,
		"ami-dff15fbc")

	h_us_west_2 := NewHubMonitor("us-west-2", 2,
		RM_HUB_HOSTS_URI,
		"ami-cf4df8af")
	h_us_east_1 := NewHubMonitor("us-east-1", 1,
		RM_HUB_HOSTS_URI,
		"ami-f31306e4")
	h_eu_west_1 := NewHubMonitor("eu-west-1", 1,
		RM_HUB_HOSTS_URI,
		"ami-66240415")
	h_ap_southeast_1 := NewHubMonitor("ap-southeast-1", 1,
		RM_HUB_HOSTS_URI,
		"ami-dff15fbc")

	for {
		log.Println("***************************************")
		log.Println("checking us-west-2")
		log.Println("***************************************")
		m_us_west_2.update()
		h_us_west_2.update()

		log.Println("***************************************")
		log.Println("checking us-east-1")
		log.Println("***************************************")
		m_us_east_1.update()
		h_us_east_1.update()

		log.Println("***************************************")
		log.Println("checking eu-west-1")
		log.Println("***************************************")
		m_eu_west_1.update()
		h_eu_west_1.update()

		log.Println("***************************************")
		log.Println("checking ap-southeast-1")
		log.Println("***************************************")
		m_ap_southeast_1.update()
		h_ap_southeast_1.update()

		log.Println("***************************************")
		log.Println("creating new hosts in us-west-2")
		log.Println("***************************************")
		m_us_west_2.createHosts()
		h_us_west_2.createHosts()

		log.Println("***************************************")
		log.Println("creating new hosts in us-east-1")
		log.Println("***************************************")
		m_us_east_1.createHosts()
		h_us_east_1.createHosts()

		log.Println("***************************************")
		log.Println("creating new hosts in eu-west-1")
		log.Println("***************************************")
		m_eu_west_1.createHosts()
		h_eu_west_1.createHosts()

		log.Println("***************************************")
		log.Println("creating new hosts in ap-southeast-1")
		log.Println("***************************************")
		m_ap_southeast_1.createHosts()
		h_ap_southeast_1.createHosts()

		log.Println("***************************************")
		log.Println("I am waiting AWS to populate IP for 5s.")
		log.Println("***************************************")
		time.Sleep(time.Second * 5)

		log.Println("***************************************")
		log.Println("provisioning us-west-2")
		log.Println("***************************************")
		m_us_west_2.provision()
		h_us_west_2.provision()

		log.Println("***************************************")
		log.Println("provisioning us-east-1")
		log.Println("***************************************")
		m_us_east_1.provision()
		h_us_east_1.provision()

		log.Println("***************************************")
		log.Println("provisioning eu-west-1")
		log.Println("***************************************")
		m_eu_west_1.provision()
		h_eu_west_1.provision()

		log.Println("***************************************")
		log.Println("provisioning ap-southeast-1")
		log.Println("***************************************")
		m_ap_southeast_1.provision()
		h_ap_southeast_1.provision()

		log.Println("***************************************")
		log.Println("I am waiting to terminate hosts for 5s.")
		log.Println("***************************************")
		time.Sleep(time.Second * 5)

		log.Println("***************************************")
		log.Println("terminating unused hosts in us-west-2")
		log.Println("***************************************")
		m_us_west_2.deleteHosts()
		h_us_west_2.deleteHosts()

		log.Println("***************************************")
		log.Println("terminating unused hosts in us-east-1")
		log.Println("***************************************")
		m_us_east_1.deleteHosts()
		h_us_east_1.deleteHosts()

		log.Println("***************************************")
		log.Println("terminating unused hosts in eu-west-1")
		log.Println("***************************************")
		m_eu_west_1.deleteHosts()
		h_eu_west_1.deleteHosts()

		log.Println("***************************************")
		log.Println("terminating unused hosts in ap-southeast-1")
		log.Println("***************************************")
		m_ap_southeast_1.deleteHosts()
		h_ap_southeast_1.deleteHosts()

	}
}
