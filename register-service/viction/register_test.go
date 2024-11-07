package register

import "testing"

func TestVictionRegisterService(t *testing.T) {
	node := &VicNode{}

	RegisterService(node, "TestService")

	err := node.Start()
	if err != nil {
		t.Fatalf("Failed to start node: %v", err)
	}

	if len(node.services) != 1 {
		t.Errorf("Expected 1 services, but got %d", len(node.services))
	}

	for _, service := range node.services {
		sampleService, ok := service.(*VicService)
		if !ok {
			t.Errorf("Failed to cast service to SampleService")
		}
		if !sampleService.started {
			t.Errorf("Expected service %s to be started, but it was not", sampleService.name)
		}
	}

	err = node.Stop()
	if err != nil {
		t.Fatalf("Failed to stop node: %v", err)
	}

	for _, service := range node.services {
		sampleService, ok := service.(*VicService)
		if !ok {
			t.Errorf("Failed to cast service to MockService")
		}
		if !sampleService.stopped {
			t.Errorf("Expected service %s to be stopped, but it was not", sampleService.name)
		}
	}
}
