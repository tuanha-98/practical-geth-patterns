package register

import (
	"reflect"
)

// ----------------------
// node/service.go
type ServiceContext struct {
	services map[reflect.Type]Service
}

type ServiceConstructor func(ctx *ServiceContext) (Service, error)

type Service interface {
	Start() error
	Stop() error
}

// ----------------------
// node/node.go
type VicNode struct {
	serviceFuncs []ServiceConstructor
	services     map[reflect.Type]Service
}

func (v *VicNode) Register(constructor ServiceConstructor) error {
	v.serviceFuncs = append(v.serviceFuncs, constructor)
	return nil
}

func (v *VicNode) Start() error {
	services := make(map[reflect.Type]Service)
	for _, constructor := range v.serviceFuncs {
		ctx := &ServiceContext{
			services: make(map[reflect.Type]Service),
		}

		for kind, s := range services {
			ctx.services[kind] = s
		}

		service, err := constructor(ctx)
		if err != nil {
			return err
		}
		kind := reflect.TypeOf(service)

		services[kind] = service
	}

	for _, service := range services {
		if err := service.Start(); err != nil {
			return err
		}
	}

	v.services = services
	return nil
}

func (v *VicNode) Stop() error {
	for _, service := range v.services {
		if err := service.Stop(); err != nil {
			return err
		}
	}
	return nil
}

// ----------------------
// eth/backend.go
type VicService struct {
	name    string
	started bool
	stopped bool
}

func New(ctx *ServiceContext, name string) (*VicService, error) {
	service := &VicService{name: name}
	return service, nil
}

func (s *VicService) Start() error {
	s.started = true
	return nil
}

func (s *VicService) Stop() error {
	s.stopped = true
	return nil
}

// ----------------------
// cmd/utils/utils.go
func RegisterService(stack *VicNode, name string) {
	stack.Register(func(ctx *ServiceContext) (Service, error) {
		return New(ctx, name)
	})
}
