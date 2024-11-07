package register

// ------------------------
// node/lifecycle.go
type Lifecycle interface {
	Start() error
	Stop() error
}

// ------------------------
// node/node.go
type GethNode struct {
	lifecycles []Lifecycle
}

func (g *GethNode) RegisterLifecycle(lifecycle Lifecycle) {
	g.lifecycles = append(g.lifecycles, lifecycle)
}

func (g *GethNode) Start() {

	lifecycles := make([]Lifecycle, len(g.lifecycles))
	copy(lifecycles, g.lifecycles)

	for _, lifecycle := range lifecycles {
		if err := lifecycle.Start(); err != nil {
			break
		}
	}
}

func (g *GethNode) stopServices(running []Lifecycle) error {
	for i := len(running) - 1; i >= 0; i-- {
		if err := running[i].Stop(); err != nil {
			return err
		}
	}
	return nil
}

// ------------------------
// eth/backend.go
type GethService struct {
	name    string
	started bool
	stopped bool
}

func New(stack *GethNode, name string) (*GethService, error) {
	service := &GethService{name: name}
	stack.RegisterLifecycle(service)
	return service, nil
}

func (s *GethService) Start() error {
	s.started = true
	return nil
}

func (s *GethService) Stop() error {
	s.stopped = true
	return nil
}

// ------------------------
// cmd/utils/flags.go
func RegisterService(stack *GethNode, name string) (*GethService, error) {
	return New(stack, name)
}
