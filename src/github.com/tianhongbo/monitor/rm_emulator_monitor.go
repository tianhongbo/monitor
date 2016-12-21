package main

import (
	"bytes"
	"encoding/json"
	//	"errors"
	"github.com/antonholmquist/jason"
	"log"
	"math"
	"net/http"
	//	"strconv"
	//"time"
)

var emulator_ports = [...]string{
	"5555",
	"5557",
	"5559",
	"5561",
	"5563",
	"5565",
	"5567",
	"5569",
	"5571",
	"5573",
}

type emulator_t struct {
	Status  string `json:"status"`  //"available",
	Adb_uri string `json:"adb_uri"` //"0.0.0.0:0",
	Region  string `json:"region"`  //"us-west-2"
}

type emulator_host_t struct {
	Status    string   `json:"status"`
	Uri       string   `json:"uri"`
	Region    string   `json:"region"`
	Aws_id    string   `json:"aws_id"`
	Emulators []string `json:"emulators"`
}

type emulator_monitor_t struct {
	Region                    string
	Threshhold                int
	Image_id                  string
	Emulators_uri             string
	Hosts_uri                 string
	Total_hosts               int
	Total_emulators           int
	Total_available_emulators int
	NewHostsList              []*ec2_t
	UnusedHostsList           []*ec2_t
}

func init() {

}

func NewEmulatorMonitor(region string, threshhold int, emu_uri string, host_uri string, image_id string) *emulator_monitor_t {
	return &emulator_monitor_t{Region: region,
		Threshhold:    threshhold,
		Emulators_uri: emu_uri,
		Hosts_uri:     host_uri,
		Image_id:      image_id}
}

func (m *emulator_monitor_t) updateTotalNumOfAvailableEmulators() {
	uri := m.Emulators_uri
	uri = uri + "?filter[region]=" + m.Region + "&filter[status]=available"
	log.Println("send update emulators total numbers request to uri: ", uri)
	resp, err := http.Get(uri)
	if err != nil {
		// handle error
		log.Println("fail to get total available emulators number response. error: ", err)
		return
	}

	v, err := jason.NewObjectFromReader(resp.Body)
	if err != nil {
		// handle error
		log.Println("fail to read total available emulators number. error: ", err)
		return
	}
	total, err := v.GetNumber("total")
	if err != nil {
		// handle error
		log.Println("fail to get total available emulators number. error: ", err)
		return
	}
	num, err := total.Int64()
	if err != nil {
		// handle error
		log.Println("fail to get total available emulators number. error: ", err)
		return
	}
	m.Total_available_emulators = int(num)
	log.Println("receive total available emulators number with total: ", m.Total_available_emulators)

	return
}

func (m *emulator_monitor_t) prepareHosts() {
	if m.Total_available_emulators >= m.Threshhold {
		return
	}
	num := float64(m.Threshhold-m.Total_available_emulators) / float64(MAX_NUM_EMULATORS_PER_HOST)
	// ceil
	total := int(math.Ceil(num))
	// do not launch instances > 2 once time
	if total > 2 {
		log.Println("request too many emulator hosts. num: ", total, ", force to 2.")
		total = 2
	}
	for i := 0; i < total; i++ {

		m.NewHostsList = append(m.NewHostsList, NewEc2("us-west-2", EMULATOR_IMAGE_ID, EMULATOR_EC2_TYPE, EMULATOR_KEY_PAIR, EMULATOR_SECURITY_GROUP, EUMLATOR_TAG))
	}
	log.Println("new hosts list: ", m.NewHostsList)

}

func (m *emulator_monitor_t) update() {

	// clean up
	m.NewHostsList = m.NewHostsList[:0]
	m.UnusedHostsList = m.UnusedHostsList[:0]

	//get total num of hosts
	uri := m.Hosts_uri
	uri = uri + "?populate=emulators&filter[region]=" + m.Region + "&filter[status]=running"
	log.Println("send emulator hosts update request to uri: ", uri)
	resp, err := http.Get(uri)
	if err != nil {
		// handle error
		log.Println("fail to get emulator hosts update message. error: ", err)
		return
	}

	v, err := jason.NewObjectFromReader(resp.Body)

	total, _ := v.GetNumber("total")
	num, err := total.Int64()
	m.Total_hosts = int(num)
	m.Total_emulators = m.Total_hosts * 10
	log.Println("receive emulator hosts update response with total: ", m.Total_hosts)

	//travers each host to lookup those need to be terminated
	hosts, _ := v.GetObjectArray("payload")
	for _, host := range hosts {
		//flag = true: need to terminate, false: not terminate
		flag := true
		emus, _ := host.GetObjectArray("emulators")
		for _, emu := range emus {
			status, _ := emu.GetString("status")
			if status != RM_STATUS_TERMINATED {
				flag = false
				break
			}
		}

		id, _ := host.GetString("_id")
		aws_id, _ := host.GetString("aws_id")
		if flag {
			log.Println("host is added to unused list. aws_id: ", id)
			e := NewEc2("us-west-2", EMULATOR_IMAGE_ID, EMULATOR_EC2_TYPE, EMULATOR_KEY_PAIR, EMULATOR_SECURITY_GROUP, EUMLATOR_TAG)
			e.InstanceId = aws_id
			e.Id = id
			m.UnusedHostsList = append(m.UnusedHostsList, e)
		} else {
			log.Println("host is in use. id: ", id, " aws id: ", aws_id)
		}
	}
	log.Println("unused hosts list: ", m.UnusedHostsList)

	m.updateTotalNumOfAvailableEmulators()

	m.prepareHosts()
	return

}

func (m *emulator_monitor_t) createHosts() {

	for i, vm := range m.NewHostsList {
		err := vm.launch()
		if err != nil {
			m.NewHostsList = m.NewHostsList[:i]
			return
		}
	}
	return

}

// handle one message
func (m *emulator_monitor_t) provision() error {

	// populate IP address
	for _, vm := range m.NewHostsList {
		vm.getIpAddr()
	}

	/*
	   	Status  string `json:"status"`  //"available",
	        Adb_uri string `json:"adb_uri"` //"0.0.0.0:0",
	        Region  string `json:"region"`  //"us-west-2"
	*/
	emu := emulator_t{Status: RM_STATUS_AVAILABLE, Region: m.Region}

	// send emualtor one by one
	for _, vm := range m.NewHostsList {

		// initialize emulator host req for storing emulator id
		emu_host := emulator_host_t{}

		for _, port := range emulator_ports {
			emu.Adb_uri = vm.PublicIp + ":" + port
			// code to json
			b := new(bytes.Buffer)
			json.NewEncoder(b).Encode(emu)
			resp, err := http.Post(m.Emulators_uri, "application/json", b)
			if err != nil {
				log.Println("fail to post one emulator. info: ", emu)
			}
			log.Println("post one emulator. info: ", emu)
			log.Println("resp: ", resp)
			v, err := jason.NewObjectFromReader(resp.Body)
			if err != nil {
				// handle error
				log.Println("fail to read post emulator rsp. error: ", err)
			}
			id, err := v.GetString("payload", "_id")
			if err != nil {
				// handle error
				log.Println("fail to get emulator id from post emulator rsp. error: ", err)
			}

			emu_host.Emulators = append(emu_host.Emulators, id)
		}

		// send emulator host

		emu_host.Status = RM_STATUS_RUNNING
		emu_host.Uri = vm.PublicIp
		emu_host.Region = m.Region
		emu_host.Aws_id = vm.InstanceId

		// code to json
		b := new(bytes.Buffer)
		json.NewEncoder(b).Encode(emu_host)

		// send POST request
		resp, err := http.Post(m.Hosts_uri, "application/json", b)
		if err != nil {
			log.Println("fail to post one emulator host. error: ", err)
		}
		log.Println("post one emulator host. info: ", emu_host)
		log.Println("post emulator host rsp: ", resp)

	}

	return nil
}

// delete hosts
func (m *emulator_monitor_t) deleteHosts() {

	for _, vm := range m.UnusedHostsList {
		vm.terminate()
		m.terminateHost(vm)
	}

	return

}

func (m *emulator_monitor_t) terminateHost(vm *ec2_t) {

	// send emulator host
	emu_host := emulator_host_t{}

	emu_host.Status = RM_STATUS_TERMINATED
	emu_host.Uri = vm.PublicIp
	emu_host.Region = m.Region
	emu_host.Aws_id = vm.InstanceId

	// code to json
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(emu_host)

	// set url
	url := RM_EMULATOR_HOSTS_URI + "/" + vm.Id

	// send PUT request
	client := &http.Client{}
	request, err := http.NewRequest("PUT", url, b)
	request.Header.Set("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("send terminate emulator req to: ", url)
	log.Println("receive emulator rsp: ", response)
	return
}
