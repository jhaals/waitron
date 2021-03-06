package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"waitron/config"
	"waitron/inventoryplugins"
	"waitron/machine"
	"waitron/waitron"

	"github.com/julienschmidt/httprouter"
)

// Test plugin #1
type TestPlugin struct {
}

func (t *TestPlugin) Init() error {
	return nil
}

func (t *TestPlugin) GetMachine(s string, m string) (*machine.Machine, error) {

	if s == "test01.prod" {
		return &machine.Machine{Hostname: "test01.prod", ShortName: "test01"}, nil
	}

	return nil, nil
}

func (t *TestPlugin) PutMachine(m *machine.Machine) error {
	return nil
}

func (t *TestPlugin) Deinit() error {
	return nil
}

// Test plugin #2
type TestPlugin2 struct {
}

func (t *TestPlugin2) Init() error {
	return nil
}

func (t *TestPlugin2) GetMachine(s string, m string) (*machine.Machine, error) {

	mm := &machine.Machine{
		Hostname:       "test01.prod",
		ShortName:      "test02",
		Domain:         "domain02",
		IpmiAddressRaw: "original_ipmi_address",
		Network: []machine.Interface{
			machine.Interface{
				MacAddress: "de:ad:be:ef",
			},
		},
	}

	if s == "test01.prod" {
		return mm, nil
	}

	return nil, nil
}

func (t *TestPlugin2) PutMachine(m *machine.Machine) error {
	return nil
}

func (t *TestPlugin2) Deinit() error {
	return nil
}
func TestPixieHandlerNotInBuildMode(t *testing.T) {

	cf := &config.Config{
		BuildType: config.BuildType{
			Cmdline:  "cmd",
			ImageURL: "image.com",
			Kernel:   "popcorn",
			Initrd:   []string{"initrd"},
		},
		MachineInventoryPlugins: []config.MachineInventoryPluginSettings{
			config.MachineInventoryPluginSettings{
				Name: "test1",
				Type: "test1",
			},
			config.MachineInventoryPluginSettings{
				Name: "test2",
				Type: "test2",
			},
		},
	}

	w := waitron.New(cf)

	/************** Stand up **************/
	if err := inventoryplugins.AddMachineInventoryPlugin("test1", func(s *config.MachineInventoryPluginSettings, c *config.Config, lf func(string, config.LogLevel) bool) inventoryplugins.MachineInventoryPlugin {
		return &TestPlugin{}
	}); err != nil {
		t.Errorf("Plugin factory failed to add test1 type: %v", err)
		return
	}

	if err := inventoryplugins.AddMachineInventoryPlugin("test2", func(s *config.MachineInventoryPluginSettings, c *config.Config, lf func(string, config.LogLevel) bool) inventoryplugins.MachineInventoryPlugin {
		return &TestPlugin2{}
	}); err != nil {
		t.Errorf("Plugin factory failed to add test1 type: %v", err)
		return
	}

	if err := w.Init(); err != nil {
		t.Errorf("Failed to init: %v", err)
		return
	}

	if err := w.Run(); err != nil {
		t.Errorf("Failed to run: %v", err)
		return
	}

	/******************************************************************/

	request, _ := http.NewRequest("PUT", "/boot", nil)
	request.Body = ioutil.NopCloser(bytes.NewBufferString("{\"ipmi_address\": \"new_ipmi_address\"}"))
	response := httptest.NewRecorder()
	ps := httprouter.Params{httprouter.Param{Key: "hostname", Value: "test01.prod"}}
	buildHandler(response, request, ps, w)

	expected := "\"State\":\"OK\"}"
	if !strings.Contains(response.Body.String(), expected) {
		t.Errorf("Reponse body is '%s', expected to contain '%s'", response.Body, expected)
	}

	request, _ = http.NewRequest("GET", "/status", nil)
	response = httptest.NewRecorder()
	ps = httprouter.Params{}
	status(response, request, ps, w)

	expected = "new_ipmi_address"
	if !strings.Contains(response.Body.String(), expected) {
		t.Errorf("Reponse body is '%s', expected to contain '%s'", response.Body, expected)
	}

	request, _ = http.NewRequest("GET", "/boot", nil)
	response = httptest.NewRecorder()
	ps = httprouter.Params{httprouter.Param{Key: "macaddr", Value: "cow"}}

	pixieHandler(response, request, ps, w)

	expected = "failed to get pxe config"
	if !strings.Contains(response.Body.String(), expected) {
		t.Errorf("Reponse body is '%s', expected to contain '%s'", response.Body, expected)
	}

}
