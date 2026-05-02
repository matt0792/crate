package internal

import (
	"time"

	"github.com/google/uuid"
)

type Identifier struct {
	Id        string
	Namespace string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewIdentifier(namespace string) *Identifier {
	now := time.Now().UTC()
	return &Identifier{
		Id:        uuid.NewString(),
		Namespace: namespace,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

type IdentifiedObject struct {
	*Identifier
	Data []byte
}

func NewIdentifiedObject(namespace string, data []byte) IdentifiedObject {
	return IdentifiedObject{
		Identifier: NewIdentifier(namespace),
		Data:       data,
	}
}

type namespace struct {
	namespace string
	objects   map[string]IdentifiedObject
}

func NewNamespace(name string) *namespace {
	return &namespace{
		namespace: name,
		objects:   map[string]IdentifiedObject{},
	}
}

func (n *namespace) Count() int {
	return len(n.objects)
}
