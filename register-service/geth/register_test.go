package register

import (
	"testing"
)

func TestGethRegisterService(t *testing.T) {
	node := &GethNode{}

	sampleService := &GethService{name: "TestService"}
	node.RegisterLifecycle(sampleService)

	if len(node.lifecycles) != 1 {
		t.Errorf("Expected 1 services, but got %d", len(node.lifecycles))
	}

	node.Start()

	if !sampleService.started {
		t.Errorf("Expected service %s to be started, but it was not", sampleService.name)
	}

	err := node.stopServices(node.lifecycles)
	if err != nil {
		t.Errorf("Error stopping services: %v", err)
	}

	if !sampleService.stopped {
		t.Errorf("Expected service %s to be stopped, but it was not", sampleService.name)
	}
}
