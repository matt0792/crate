package crate

import "github.com/matt0792/crate/internal"

// Project initialises crate. Must be called on startup.
// Reads the wal into memory & compacts it.
func Project(name string) error {
	return internal.Project(name)
}

// Size returns the project file size on disk, in bytes.
func Size() (int64, error) {
	return internal.Size()
}

// Count returns the total amount of items in a namespace.
// Returns 0 if the namespace does not exist.
func Count(namespace string) int {
	return internal.Count(namespace)
}

// Store stores an object in a namespace, and returns its id.
// Namespace will be created if it does not exist.
func Store[T any](namespace string, obj T) (string, error) {
	return internal.Store(namespace, obj)
}

// Update updates an object by its id.
// Returns a boolean indicating success.
func Update[T any](namespace, id string, obj T) bool {
	return internal.Update(namespace, id, obj)
}

// Get retrieves an object by id.
func Get[T any](namespace, id string) (T, bool) {
	return internal.Get[T](namespace, id)
}

func Find[T any](namespace string, filter func(obj T) bool) []T {
	return internal.Find(namespace, filter)
}

func FindPaged[T any](namespace string, skip, limit int, filter func(obj T) bool) []T {
	return internal.FindPaged(namespace, skip, limit, filter)
}

func Delete(namespace, id string) {
	internal.Delete(namespace, id)
}

func DeleteBy[T any](namespace string, filter func(obj T) bool) int {
	return internal.DeleteBy(namespace, filter)
}

func Namespaces() []string {
	return internal.Namespaces()
}
