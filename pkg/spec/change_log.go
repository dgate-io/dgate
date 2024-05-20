package spec

import (
	"strconv"
	"strings"
	"time"
)

type ChangeLog struct {
	ID        string  `json:"id"`
	Cmd       Command `json:"cmd"`
	Name      string  `json:"name"`
	Namespace string  `json:"namespace"`
	Item      any     `json:"item"`
	Version   int     `json:"version"`
	errChan   chan error
}

func NewChangeLog(item Named, namespace string, cmd Command) *ChangeLog {
	if item == nil {
		panic("item cannot be nil")
	}
	if namespace == "" {
		panic("namespace cannot be empty")
	}
	if cmd.String() == "" {
		panic("cmd cannot be empty")
	}
	return &ChangeLog{
		Version:   1,
		ID:        strconv.FormatInt(time.Now().UnixNano(), 36),
		Cmd:       cmd,
		Item:      item,
		Name:      item.GetName(),
		Namespace: namespace,
	}
}

func (cl *ChangeLog) SetErrorChan(errChan chan error) {
	cl.errChan = errChan
}

func (cl *ChangeLog) PushError(err error) {
	if cl.errChan != nil {
		cl.errChan <- err
	}
}

type Command string

type Action string
type Resource string

const (
	Add    Action = "add"
	Delete Action = "delete"

	Routes      Resource = "route"
	Services    Resource = "service"
	Namespaces  Resource = "namespace"
	Modules     Resource = "module"
	Domains     Resource = "domain"
	Collections Resource = "collection"
	Documents   Resource = "document"
	Secrets     Resource = "secret"
)

var (
	AddRouteCommand         Command = newCommand(Add, Routes)
	AddServiceCommand       Command = newCommand(Add, Services)
	AddNamespaceCommand     Command = newCommand(Add, Namespaces)
	AddModuleCommand        Command = newCommand(Add, Modules)
	AddDomainCommand        Command = newCommand(Add, Domains)
	AddCollectionCommand    Command = newCommand(Add, Collections)
	AddDocumentCommand      Command = newCommand(Add, Documents)
	AddSecretCommand        Command = newCommand(Add, Secrets)
	DeleteRouteCommand      Command = newCommand(Delete, Routes)
	DeleteServiceCommand    Command = newCommand(Delete, Services)
	DeleteNamespaceCommand  Command = newCommand(Delete, Namespaces)
	DeleteModuleCommand     Command = newCommand(Delete, Modules)
	DeleteDomainCommand     Command = newCommand(Delete, Domains)
	DeleteCollectionCommand Command = newCommand(Delete, Collections)
	DeleteDocumentCommand   Command = newCommand(Delete, Documents)
	DeleteSecretCommand     Command = newCommand(Delete, Secrets)

	// internal commands
	NoopCommand     Command = Command("noop")
	ShutdownCommand Command = Command("shutdown")
	RestartCommand  Command = Command("restart")
)

func newCommand(action Action, resource Resource) Command {
	return Command(action.String() + "_" + resource.String())
}

func (act Action) String() string {
	return string(act)
}

func (rt Resource) String() string {
	return string(rt)
}

func (clc Command) String() string {
	if clc.IsNoop() {
		return "noop"
	}
	return string(clc)
}

func (clc Command) IsNoop() bool {
	return string(clc) == "noop"
}

func (resource1 Resource) IsRelatedTo(resource2 Resource) bool {
	if resource1 == resource2 || resource1 == Namespaces || resource2 == Namespaces {
		return true
	}
	switch resource1 {
	case Routes:
		return resource2 == Services || resource2 == Modules
	case Services:
		return resource2 == Routes
	case Modules:
		return resource2 == Namespaces
	case Collections:
		return resource2 == Documents
	case Documents:
		return resource2 == Collections
	default:
		return false
	}
}

func (clc Command) Action() Action {
	if strings.HasPrefix(string(clc), "add_") {
		return Add
	} else if strings.HasPrefix(string(clc), "delete_") {
		return Delete
	} else if clc.IsNoop() {
		return "noop"
	}
	panic("change log: invalid command")
}

func (clc Command) Resource() Resource {
	if clc.IsNoop() {
		return "noop"
	}
	cmdString := string(clc)
	switch {
	case strings.HasSuffix(cmdString, "_route"):
		return Routes
	case strings.HasSuffix(cmdString, "_service"):
		return Services
	case strings.HasSuffix(cmdString, "_namespace"):
		return Namespaces
	case strings.HasSuffix(cmdString, "_module"):
		return Modules
	case strings.HasSuffix(cmdString, "_domain"):
		return Domains
	case strings.HasSuffix(cmdString, "_collection"):
		return Collections
	case strings.HasSuffix(cmdString, "_document"):
		return Documents
	case strings.HasSuffix(cmdString, "_secret"):
		return Secrets
	default:
		panic("change log: invalid command")
	}
}
