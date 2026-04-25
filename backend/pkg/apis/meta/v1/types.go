package v1

import (
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var resourceNamePattern = regexp.MustCompile(`^[a-z0-9_-]+$`)

type TypeMeta struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
}

type ObjectMeta struct {
	Name              string            `json:"name,omitempty"`
	UID               string            `json:"uid,omitempty"`
	Labels            map[string]string `json:"labels,omitempty"`
	Annotations       map[string]string `json:"annotations,omitempty"`
	Generation        int64             `json:"generation,omitempty"`
	ResourceVersion   string            `json:"resourceVersion,omitempty"`
	CreationTimestamp time.Time         `json:"creationTimestamp,omitempty"`
}

type ListMeta struct {
	Continue        string `json:"continue,omitempty"`
	ResourceVersion string `json:"resourceVersion,omitempty"`
}

type Object[Spec any, Status any] struct {
	TypeMeta `json:",inline"`
	Metadata ObjectMeta `json:"metadata"`
	Spec     Spec       `json:"spec"`
	Status   Status     `json:"status,omitempty"`
}

type List[T any] struct {
	TypeMeta `json:",inline"`
	Metadata ListMeta `json:"metadata"`
	Items    []T      `json:"items"`
}

func (m *ObjectMeta) Bind(_ *http.Request) error {
	m.Name = strings.TrimSpace(m.Name)
	if m.Name != "" && !resourceNamePattern.MatchString(m.Name) {
		return errors.New("metadata.name must match ^[a-z0-9_-]+$")
	}
	if m.Labels == nil {
		m.Labels = map[string]string{}
	}
	if m.Annotations == nil {
		m.Annotations = map[string]string{}
	}
	return nil
}

func (l *List[T]) Bind(_ *http.Request) error {
	if l.Items == nil {
		l.Items = []T{}
	}
	return nil
}
