package main

import (
	"bytes"
	"encoding/json"
	//	"errors"
	"github.com/antonholmquist/jason"
	"log"
	//"math"
	"net/http"
	//	"strconv"
	//"time"
)

type hub_host_t struct {
	Status string `json:"status"`
	Uri    string `json:"uri"`
	Region string `json:"region"`
	Aws_id string `json:"aws_id"`
}

type hub_monitor_t struct {
	Region               string
	Threshhold           int
	Image_id             string
	Hosts_uri            string
	Total_hosts          int
	Total_available_hubs int
	NewHostsList         []*ec2_t
	UnusedHostsList      []*ec2_t
}

func init() {

}

func NewHubMonitor(region string, threshhold int, host_uri string, image_id string) *hub_monitor_t {
	return &hub_monitor_t{Region: region,
		Threshhold: threshhold,
		Hosts_uri:  host_uri,
		Image_id:   image_id}
}

func (m *hub_monitor_t) updateTotalNumOfAvailableHubs() {
	uri := m.Hosts_uri
	uri = uri + "?filter[region]=" + m.Region + "&filter[status]=available"
	log.Println("send update hub hosts total numbers request to uri: ", uri)
	resp, err := http.Get(uri)
	if err != nil {
		// handle error
		log.Println("fail to get total available hubs number response. error: ", err)
		return
	}
	defer resp.Body.Close()
	v, err := jason.NewObjectFromReader(resp.Body)
	if err != nil {
		// handle error
		log.Println("fail to read total available hubs number. error: ", err)
		return
	}
	total, err := v.GetNumber("total")
	if err != nil {
		// handle error
		log.Println("fail to get total available hubs number. error: ", err)
		return
	}
	num, err := total.Int64()
	if err != nil {
		// handle error
		log.Println("fail to get total available hubs number. error: ", err)
		return
	}
	m.Total_available_hubs = int(num)
	log.Println("receive total available hubs num: ", m.Total_available_hubs)

	return
}

func (m *hub_monitor_t) prepareHosts() {
	if m.Total_available_hubs >= m.Threshhold {
		return
	}
	num := m.Threshhold - m.Total_available_hubs
	//
	for i := 0; i < num; i++ {

		m.NewHostsList = append(m.NewHostsList, NewEc2("us-west-2", HUB_IMAGE_ID, HUB_EC2_TYPE, HUB_KEY_PAIR, HUB_SECURITY_GROUP, HUB_TAG))
	}
	log.Println("new hub hosts list: ", m.NewHostsList)

}

func (m *hub_monitor_t) update() {

	// clean up
	m.NewHostsList = m.NewHostsList[:0]
	m.UnusedHostsList = m.UnusedHostsList[:0]

	//get total num of hosts
	uri := m.Hosts_uri
	uri = uri + "?filter[region]=" + m.Region + "&filter[status]=terminating"
	log.Println("send hub hosts update request to uri: ", uri)
	resp, err := http.Get(uri)
	if err != nil {
		// handle error
		log.Println("fail to get hub hosts update message. error: ", err)
		return
	}
	defer resp.Body.Close()
	v, err := jason.NewObjectFromReader(resp.Body)

	total, _ := v.GetNumber("total")
	num, err := total.Int64()
	log.Println("receive hub hosts update response with total: ", num)

	//travers each host to lookup those need to be terminated
	hosts, _ := v.GetObjectArray("payload")
	for _, host := range hosts {

		id, _ := host.GetString("_id")
		aws_id, _ := host.GetString("aws_id")
		e := NewEc2("us-west-2", HUB_IMAGE_ID, HUB_EC2_TYPE, HUB_KEY_PAIR, HUB_SECURITY_GROUP, HUB_TAG)
		e.InstanceId = aws_id
		e.Id = id
		m.UnusedHostsList = append(m.UnusedHostsList, e)
		log.Println("host is in use. id: ", id, " aws id: ", aws_id)
	}
	log.Println("unused hub hosts list: ", m.UnusedHostsList)

	m.updateTotalNumOfAvailableHubs()

	m.prepareHosts()
	return

}

func (m *hub_monitor_t) createHosts() {

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
func (m *hub_monitor_t) provision() error {

	// populate IP address
	for _, vm := range m.NewHostsList {
		vm.getIpAddr()
	}

	// send emualtor one by one
	for _, vm := range m.NewHostsList {

		// initialize emulator host req for storing emulator id
		hub_host := hub_host_t{}

		// send emulator host
		hub_host.Status = RM_STATUS_AVAILABLE
		hub_host.Uri = vm.PublicIp
		hub_host.Region = m.Region
		hub_host.Aws_id = vm.InstanceId

		// code to json
		b := new(bytes.Buffer)
		json.NewEncoder(b).Encode(hub_host)

		// send POST request
		resp, err := http.Post(m.Hosts_uri, "application/json", b)
		if err != nil {
			log.Println("fail to post one hub host. error: ", err)
		}
		defer resp.Body.Close()
		log.Println("post one hub host. info: ", hub_host)
		log.Println("post emulator host rsp: ", resp)

	}

	return nil
}

// delete hosts
func (m *hub_monitor_t) deleteHosts() {

	for _, vm := range m.UnusedHostsList {
		vm.terminate()
		m.terminateHost(vm)
	}

	return

}

func (m *hub_monitor_t) terminateHost(vm *ec2_t) {

	// send emulator host
	hub_host := hub_host_t{}

	hub_host.Status = RM_STATUS_TERMINATED
	hub_host.Uri = vm.PublicIp
	hub_host.Region = m.Region
	hub_host.Aws_id = vm.InstanceId

	// code to json
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(hub_host)

	// set url
	url := RM_HUB_HOSTS_URI + "/" + vm.Id

	// send PUT request
	client := &http.Client{}
	request, err := http.NewRequest("PUT", url, b)
	request.Header.Set("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		log.Println(err)
		return
	}
	defer response.Body.Close()
	log.Println("send terminate hub host req to: ", url)
	log.Println("receive terminate hub host rsp: ", response)
	return
}
