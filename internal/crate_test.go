package internal

import (
	"fmt"
	"sync"
	"testing"
)

const testProjectName = "cratetest"

// --- shared test types ---

type TestObject struct {
	TestField       string
	unexportedField int
}

type NestedTO struct {
	Id  string
	Obj TestObject
}

type NestedPointer struct {
	Nested *NestedTO
}

// --- helpers ---

func setup(t *testing.T) {
	t.Helper()
	Project(testProjectName)
}

// --- Store ---

func TestStore(t *testing.T) {
	setup(t)

	t.Run("simple object returns non-empty id", func(t *testing.T) {
		id, err := Store("testStore", TestObject{TestField: "foo", unexportedField: 5})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id == "" {
			t.Error("id should not be empty")
		}
	})

	t.Run("nested struct", func(t *testing.T) {
		id, err := Store("testStore", NestedTO{
			Id:  "inner-id",
			Obj: TestObject{TestField: "bar"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id == "" {
			t.Error("id should not be empty")
		}
	})

	t.Run("nested pointer struct", func(t *testing.T) {
		id, err := Store("testStore", NestedPointer{
			Nested: &NestedTO{
				Id:  "ptr-id",
				Obj: TestObject{TestField: "baz"},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id == "" {
			t.Error("id should not be empty")
		}
	})

	t.Run("nil pointer field", func(t *testing.T) {
		id, err := Store("testStore", NestedPointer{Nested: nil})
		if err != nil {
			t.Fatalf("unexpected error storing nil pointer: %v", err)
		}
		if id == "" {
			t.Error("id should not be empty")
		}
	})

	t.Run("different namespaces produce different ids", func(t *testing.T) {
		obj := TestObject{TestField: "same"}
		id1, _ := Store("nsA", obj)
		id2, _ := Store("nsB", obj)
		if id1 == id2 {
			t.Error("ids from different namespaces should not collide")
		}
	})

	t.Run("each store call produces a unique id", func(t *testing.T) {
		obj := TestObject{TestField: "dup"}
		id1, _ := Store("uniqueTest", obj)
		id2, _ := Store("uniqueTest", obj)
		if id1 == id2 {
			t.Error("storing identical objects twice should produce distinct ids")
		}
	})
}

// --- Get ---

func TestGet(t *testing.T) {
	setup(t)

	t.Run("round-trips stored value", func(t *testing.T) {
		original := TestObject{TestField: "roundtrip"}
		id, err := Store("getRoundtrip", original)
		if err != nil {
			t.Fatalf("store: %v", err)
		}

		got, ok := Get[TestObject]("getRoundtrip", id)
		if !ok {
			t.Fatal("expected to find object")
		}
		if got.TestField != original.TestField {
			t.Errorf("got %q, want %q", got.TestField, original.TestField)
		}
	})

	t.Run("missing id returns zero value and false", func(t *testing.T) {
		got, ok := Get[TestObject]("getRoundtrip", "does-not-exist")
		if ok {
			t.Error("expected ok=false for missing id")
		}
		if got.TestField != "" {
			t.Error("expected zero value on miss")
		}
	})

	t.Run("missing namespace returns false", func(t *testing.T) {
		_, ok := Get[TestObject]("no-such-namespace", "any-id")
		if ok {
			t.Error("expected ok=false for unknown namespace")
		}
	})

	t.Run("nested pointer round-trips", func(t *testing.T) {
		original := NestedPointer{
			Nested: &NestedTO{Id: "x", Obj: TestObject{TestField: "deep"}},
		}
		id, _ := Store("getPointer", original)
		got, ok := Get[NestedPointer]("getPointer", id)
		if !ok {
			t.Fatal("expected to find object")
		}
		if got.Nested == nil {
			t.Fatal("nested pointer should not be nil after round-trip")
		}
		if got.Nested.Obj.TestField != "deep" {
			t.Errorf("got %q, want %q", got.Nested.Obj.TestField, "deep")
		}
	})
}

// --- Update ---

func TestUpdate(t *testing.T) {
	setup(t)

	t.Run("updates existing object", func(t *testing.T) {
		id, _ := Store("updateNS", TestObject{TestField: "before"})
		ok := Update("updateNS", id, TestObject{TestField: "after"})
		if !ok {
			t.Fatal("expected Update to return true")
		}
		got, _ := Get[TestObject]("updateNS", id)
		if got.TestField != "after" {
			t.Errorf("got %q, want %q", got.TestField, "after")
		}
	})

	t.Run("returns false for unknown id", func(t *testing.T) {
		setup(t)
		ok := Update("updateNS", "ghost-id", TestObject{TestField: "x"})
		if ok {
			t.Error("expected false for unknown id")
		}
	})

	t.Run("returns false for unknown namespace", func(t *testing.T) {
		ok := Update("no-such-ns", "any-id", TestObject{})
		if ok {
			t.Error("expected false for unknown namespace")
		}
	})
}

// --- Delete ---

func TestDelete(t *testing.T) {
	setup(t)

	t.Run("deleted object is no longer retrievable", func(t *testing.T) {
		id, _ := Store("deleteNS", TestObject{TestField: "gone"})
		Delete("deleteNS", id)
		_, ok := Get[TestObject]("deleteNS", id)
		if ok {
			t.Error("expected object to be deleted")
		}
	})

	t.Run("deleting non-existent id does not panic", func(t *testing.T) {
		Delete("deleteNS", "phantom-id")
	})
}

// --- DeleteBy ---

func TestDeleteBy(t *testing.T) {
	setup(t)

	t.Run("removes matching objects and returns count", func(t *testing.T) {
		Store("deleteByNS", TestObject{TestField: "keep"})
		Store("deleteByNS", TestObject{TestField: "remove"})
		Store("deleteByNS", TestObject{TestField: "remove"})

		n := DeleteBy("deleteByNS", func(obj TestObject) bool {
			return obj.TestField == "remove"
		})
		if n != 2 {
			t.Errorf("expected 2 deletions, got %d", n)
		}

		remaining := Find("deleteByNS", func(obj TestObject) bool { return true })
		for _, r := range remaining {
			if r.TestField == "remove" {
				t.Error("deleted object still present")
			}
		}
	})

	t.Run("returns 0 for unknown namespace", func(t *testing.T) {
		n := DeleteBy("ghost-ns", func(obj TestObject) bool { return true })
		if n != 0 {
			t.Errorf("expected 0, got %d", n)
		}
	})
}

// --- Find ---

func TestFind(t *testing.T) {
	setup(t)

	t.Run("returns only matching objects", func(t *testing.T) {
		Store("findNS", TestObject{TestField: "alpha"})
		Store("findNS", TestObject{TestField: "beta"})
		Store("findNS", TestObject{TestField: "alpha"})

		results := Find("findNS", func(obj TestObject) bool {
			return obj.TestField == "alpha"
		})
		if len(results) != 2 {
			t.Errorf("expected 2 results, got %d", len(results))
		}
	})

	t.Run("returns empty slice for no matches", func(t *testing.T) {
		results := Find("findNS", func(obj TestObject) bool {
			return obj.TestField == "no-match"
		})
		if len(results) != 0 {
			t.Errorf("expected empty, got %d", len(results))
		}
	})

	t.Run("returns empty slice for unknown namespace", func(t *testing.T) {
		results := Find("no-such-ns", func(obj TestObject) bool { return true })
		if results == nil {
			t.Error("expected non-nil empty slice")
		}
		if len(results) != 0 {
			t.Errorf("expected 0, got %d", len(results))
		}
	})
}

// --- FindPaged ---

func TestFindPaged(t *testing.T) {
	setup(t)

	const ns = "pagedNS"
	for i := range 10 {
		Store(ns, TestObject{TestField: fmt.Sprintf("item-%02d", i)})
	}

	t.Run("limit respected", func(t *testing.T) {
		results := FindPaged[TestObject](ns, 0, 3, nil)
		if len(results) != 3 {
			t.Errorf("expected 3, got %d", len(results))
		}
	})

	t.Run("skip respected", func(t *testing.T) {
		all := FindPaged[TestObject](ns, 0, 0, nil)
		skipped := FindPaged[TestObject](ns, 2, 0, nil)
		if len(skipped) != len(all)-2 {
			t.Errorf("expected %d results after skip=2, got %d", len(all)-2, len(skipped))
		}
	})

	t.Run("skip and limit combined", func(t *testing.T) {
		results := FindPaged[TestObject](ns, 2, 4, nil)
		if len(results) != 4 {
			t.Errorf("expected 4, got %d", len(results))
		}
	})

	t.Run("skip beyond total returns empty", func(t *testing.T) {
		results := FindPaged[TestObject](ns, 1000, 5, nil)
		if len(results) != 0 {
			t.Errorf("expected empty, got %d", len(results))
		}
	})

	t.Run("filter applied before paging", func(t *testing.T) {
		// store a few with a distinct marker
		const markedNS = "pagedFiltered"
		for i := range 6 {
			Store(markedNS, TestObject{TestField: fmt.Sprintf("marked-%d", i)})
		}
		Store(markedNS, TestObject{TestField: "skip-me"})

		results := FindPaged(markedNS, 0, 3, func(obj TestObject) bool {
			return obj.TestField != "skip-me"
		})
		if len(results) != 3 {
			t.Errorf("expected 3, got %d", len(results))
		}
		for _, r := range results {
			if r.TestField == "skip-me" {
				t.Error("filtered object appeared in paged result")
			}
		}
	})
}

// --- Concurrency ---

func TestConcurrentStoreAndGet(t *testing.T) {
	setup(t)

	const goroutines = 50
	ids := make([]string, goroutines)
	var wg sync.WaitGroup

	wg.Add(goroutines)
	for i := range goroutines {
		go func(i int) {
			defer wg.Done()
			id, err := Store("concNS", TestObject{TestField: fmt.Sprintf("obj-%d", i)})
			if err != nil {
				t.Errorf("goroutine %d: store error: %v", i, err)
				return
			}
			ids[i] = id
		}(i)
	}
	wg.Wait()

	for i, id := range ids {
		if id == "" {
			t.Errorf("goroutine %d produced empty id", i)
			continue
		}
		_, ok := Get[TestObject]("concNS", id)
		if !ok {
			t.Errorf("goroutine %d: object %q not found after concurrent store", i, id)
		}
	}
}

func TestConcurrentUpdateAndGet(t *testing.T) {
	setup(t)

	id, _ := Store("concUpdateNS", TestObject{TestField: "v0"})

	var wg sync.WaitGroup
	const goroutines = 20
	wg.Add(goroutines)
	for i := range goroutines {
		go func(i int) {
			defer wg.Done()
			Update("concUpdateNS", id, TestObject{TestField: fmt.Sprintf("v%d", i)})
		}(i)
	}
	wg.Wait()

	// just assert no panic / data race; value is non-deterministic
	_, ok := Get[TestObject]("concUpdateNS", id)
	if !ok {
		t.Error("object should still exist after concurrent updates")
	}
}

// --- assertInitialised ---

func TestAssertInitialisedPanicsWithoutProject(t *testing.T) {
	// reset state to simulate uninitialised call
	crateDir = ""

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when crate.Project() not called")
		}
		// re-init so subsequent tests aren't broken
		Project(testProjectName)
	}()

	Store("any", TestObject{})
}
