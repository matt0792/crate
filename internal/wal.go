package internal

import (
	"encoding/gob"
	"io"
	"log"
	"os"
	"time"
)

type opType uint8

const (
	opStore opType = iota
	opUpdate
	opDelete
)

type walEntry struct {
	Op        opType
	Namespace string
	Id        string
	Object    *IdentifiedObject
}

var (
	walCh   chan walEntry
	walDone chan struct{}
	walFile *os.File
)

func runWAL(f *os.File, enc *gob.Encoder) {
	defer f.Close()
	defer close(walDone)

	for entry := range walCh {
		if err := enc.Encode(entry); err != nil {
			log.Println("wal encode error:", err)
		}
	}
}

func compact(path string) (*os.File, *gob.Encoder, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, nil, err
	}

	enc := gob.NewEncoder(f)
	for _, ns := range namespaces {
		for _, obj := range ns.objects {
			obj := obj
			if err := enc.Encode(walEntry{
				Op:        opStore,
				Namespace: ns.namespace,
				Id:        obj.Id,
				Object:    &obj,
			}); err != nil {
				f.Close()
				return nil, nil, err
			}
		}
	}

	return f, enc, nil
}

func loadUntil(path string, time time.Time) error {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()

	dec := gob.NewDecoder(f)
	for {
		var entry walEntry
		err := dec.Decode(&entry)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if !entry.Object.CreatedAt.After(time) {
			switch entry.Op {
			case opStore:
				ns, ok := namespaces[entry.Namespace]
				if !ok {
					ns = NewNamespace(entry.Namespace)
					namespaces[entry.Namespace] = ns
				}
				obj := entry.Object
				ns.objects[entry.Id] = IdentifiedObject{
					Identifier: &Identifier{
						Id:        obj.Id,
						Namespace: obj.Namespace,
						CreatedAt: obj.CreatedAt,
						UpdatedAt: obj.UpdatedAt,
					},
					Data: obj.Data,
				}
			case opUpdate:
				if ns, ok := namespaces[entry.Namespace]; ok {
					if existing, ok := ns.objects[entry.Id]; ok {
						existing.Data = entry.Object.Data
						existing.UpdatedAt = entry.Object.UpdatedAt
						ns.objects[entry.Id] = existing
					}
				}
			case opDelete:
				if ns, ok := namespaces[entry.Namespace]; ok {
					delete(ns.objects, entry.Id)
				}
			}
		}
	}

	return nil
}

func load(path string) error {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()

	dec := gob.NewDecoder(f)
	for {
		var entry walEntry
		err := dec.Decode(&entry)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch entry.Op {
		case opStore:
			ns, ok := namespaces[entry.Namespace]
			if !ok {
				ns = NewNamespace(entry.Namespace)
				namespaces[entry.Namespace] = ns
			}
			obj := entry.Object
			ns.objects[entry.Id] = IdentifiedObject{
				Identifier: &Identifier{
					Id:        obj.Id,
					Namespace: obj.Namespace,
					CreatedAt: obj.CreatedAt,
					UpdatedAt: obj.UpdatedAt,
				},
				Data: obj.Data,
			}
		case opUpdate:
			if ns, ok := namespaces[entry.Namespace]; ok {
				if existing, ok := ns.objects[entry.Id]; ok {
					existing.Data = entry.Object.Data
					existing.UpdatedAt = entry.Object.UpdatedAt
					ns.objects[entry.Id] = existing
				}
			}
		case opDelete:
			if ns, ok := namespaces[entry.Namespace]; ok {
				delete(ns.objects, entry.Id)
			}
		}
	}

	return nil
}

func Close() {
	if walCh == nil {
		return
	}
	close(walCh)
	<-walDone
}
