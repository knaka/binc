package common

type NewManagerFn func(dir string) Manager

type Factory struct {
	PriorityWeight int
	NewManager     NewManagerFn
}

var factories []*Factory

func RegisterManagerFactory(fn NewManagerFn, priorityWeight int) {
	factories = append(factories, &Factory{
		PriorityWeight: priorityWeight,
		NewManager:     fn,
	})
}

type Manager interface {
	//Link() ([]string, error)
	CanRun(cmd string) bool
	Run(args []string) error
}
