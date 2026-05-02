package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"
)

var projectName = ""

var (
	namespaces map[string]*namespace
	mu         sync.RWMutex
)

var crateDir string
var walPath string

func Project(name string) error {
	projectName = name

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("crate: could not find home dir: %w", err)
	}

	crateDir = filepath.Join(home, ".crate", name)
	walPath = filepath.Join(crateDir, "crate.wal")

	if err := os.MkdirAll(crateDir, 0755); err != nil {
		return err
	}

	namespaces = map[string]*namespace{}

	if err := load(walPath); err != nil {
		panic(fmt.Errorf("crate: load failed: %w", err))
	}

	f, enc, err := compact(walPath)
	if err != nil {
		return fmt.Errorf("crate: compact failed: %w", err)
	}
	walFile = f

	walCh = make(chan walEntry, 256)
	walDone = make(chan struct{})
	go runWAL(walFile, enc)
	return nil
}

func Rollback(name string, time time.Time) {
	if projectName != "" {
		return
	}

	projectName = name

	home, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Errorf("crate: could not find home dir: %w", err))
	}

	crateDir = filepath.Join(home, ".crate", name)
	walPath = filepath.Join(crateDir, "crate.wal")

	if err := loadUntil(walPath, time); err != nil {
		panic(fmt.Errorf("crate: load failed: %w", err))
	}

	os.Exit(0)
}

func Size() (int64, error) {
	info, err := os.Stat(walPath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func Count(namespace string) int {
	if ns, ok := namespaces[namespace]; ok {
		return ns.Count()
	}
	return 0
}

func Store[T any](namespace string, obj T) (string, error) {
	assertInitialised()

	data, err := encode(obj)
	if err != nil {
		return "", err
	}
	crateObj := NewIdentifiedObject(namespace, data)

	mu.Lock()
	if ns, ok := namespaces[namespace]; ok {
		ns.objects[crateObj.Id] = crateObj
	} else {
		new := NewNamespace(namespace)
		new.objects[crateObj.Id] = crateObj

		namespaces[namespace] = new
	}
	mu.Unlock()

	walCh <- walEntry{
		Op:        opStore,
		Namespace: namespace,
		Id:        crateObj.Id,
		Object:    &crateObj,
	}

	return crateObj.Id, nil
}

func Update[T any](namespace, id string, obj T) bool {
	assertInitialised()

	data, err := encode(obj)
	if err != nil {
		return false
	}

	mu.Lock()

	ns, ok := namespaces[namespace]
	if !ok {
		return false
	}

	existing, ok := ns.objects[id]
	if !ok {
		return false
	}

	existing.Data = data
	ns.objects[id] = existing

	mu.Unlock()

	walCh <- walEntry{
		Op:        opUpdate,
		Namespace: namespace,
		Id:        id,
		Object:    &existing,
	}

	return true
}

func Get[T any](namespace, id string) (T, bool) {
	assertInitialised()

	var zero T

	mu.Lock()
	defer mu.Unlock()

	if ns, ok := namespaces[namespace]; ok {
		if obj, ok := ns.objects[id]; ok {
			if val, err := decode[T](obj.Data); err == nil {
				return val, ok
			}
		}
	}

	return zero, false
}

func Find[T any](namespace string, filter func(obj T) bool) []T {
	assertInitialised()

	matches := []T{}

	mu.Lock()
	defer mu.Unlock()

	ns, ok := namespaces[namespace]
	if !ok {
		return matches
	}

	for _, v := range ns.objects {
		if obj, err := decode[T](v.Data); err == nil {
			if filter(obj) {
				matches = append(matches, obj)
			}
		}
	}

	return matches
}

func FindPaged[T any](namespace string, skip, limit int, filter func(obj T) bool) []T {
	assertInitialised()

	matches := []T{}

	mu.Lock()
	defer mu.Unlock()

	if ns, ok := namespaces[namespace]; ok {
		ordered := make([]IdentifiedObject, 0, len(ns.objects))
		for _, v := range ns.objects {
			ordered = append(ordered, v)
		}
		slices.SortFunc(ordered, func(a, b IdentifiedObject) int {
			return a.CreatedAt.Compare(b.CreatedAt)
		})

		for _, v := range ordered {
			obj, err := decode[T](v.Data)
			if err != nil {
				continue
			}
			if filter != nil && !filter(obj) {
				continue
			}
			if skip > 0 {
				skip--
				continue
			}
			matches = append(matches, obj)
			if limit > 0 && len(matches) >= limit {
				break
			}
		}
	}
	return matches
}

func Delete(namespace, id string) {
	assertInitialised()

	mu.Lock()

	ns, ok := namespaces[namespace]
	if !ok {
		return
	}

	delete(ns.objects, id)

	mu.Unlock()

	walCh <- walEntry{
		Op:        opDelete,
		Namespace: namespace,
		Id:        id,
		Object:    nil,
	}
}

func DeleteBy[T any](namespace string, filter func(obj T) bool) int {
	assertInitialised()

	count := 0

	entries := []walEntry{}

	mu.Lock()

	ns, ok := namespaces[namespace]
	if !ok {
		return count
	}

	for _, v := range ns.objects {
		if obj, err := decode[T](v.Data); err == nil {
			if filter(obj) {
				count++
				delete(ns.objects, v.Id)

				entries = append(entries, walEntry{
					Op:        opDelete,
					Namespace: namespace,
					Id:        v.Id,
					Object:    nil,
				})
			}
		}
	}

	mu.Unlock()

	for _, e := range entries {
		walCh <- e
	}

	return count
}

func Namespaces() []string {
	keys := []string{}
	for k := range namespaces {
		keys = append(keys, k)
	}

	return keys
}

func assertInitialised() {
	if crateDir == "" {
		panic("crate: must call crate.Project() before using crate")
	}
}
