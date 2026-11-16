package memory

import (
	"bytes"
	"context"
	"maps"
	"slices"
	"sync"

	"go.mongodb.org/mongo-driver/v2/bson"

	"job-management-service/internal/repository"
)

// uniqueField este un camp care nu se poate repeta intre documente (SP: constrangerea "unic").
type uniqueField struct {
	name  string
	value string
}

// spec descrie un tip de document pentru collection: codul generic nu poate scrie doc.ID
// pentru un T oarecare, asa ca primeste functii care stiu sa ajunga la campurile lui T.
type spec[T any] struct {
	idOf   func(T) bson.ObjectID
	setID  func(*T, bson.ObjectID)
	unique func(T) []uniqueField // nil daca tipul nu are campuri unice
	// keepLinks copiaza listele de legaturi din documentul vechi in cel nou, la Replace:
	// PUT schimba doar campurile proprii. nil daca tipul nu are legaturi.
	keepLinks func(next *T, prev T)
}

// collection pastreaza in memorie documente de orice tip T.
type collection[T any] struct {
	// Gin trateaza fiecare cerere pe o goroutina separata, deci accesul la map trebuie sincronizat:
	// mai multe citiri pot rula in paralel (RLock), o scriere ruleaza singura (Lock).
	mu   sync.RWMutex
	docs map[bson.ObjectID]T
	spec spec[T]
}

func newCollection[T any](s spec[T]) *collection[T] {
	return &collection[T]{docs: make(map[bson.ObjectID]T), spec: s}
}

// sorted intoarce documentele in ordinea id-urilor. Un map nu are ordine, iar un ObjectID
// incepe cu momentul crearii, deci ordinea e stabila si e chiar cea a adaugarii.
func (c *collection[T]) sorted(docs []T) []T {
	slices.SortFunc(docs, func(a, b T) int {
		idA, idB := c.spec.idOf(a), c.spec.idOf(b)
		return bytes.Compare(idA[:], idB[:])
	})
	return docs
}

func (c *collection[T]) List(_ context.Context, offset, limit int) ([]T, int, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	all := c.sorted(slices.Collect(maps.Values(c.docs)))
	total := len(all)
	start := min(offset, total)
	end := min(start+limit, total)
	return all[start:end], total, nil
}

func (c *collection[T]) Get(_ context.Context, id bson.ObjectID) (T, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	doc, ok := c.docs[id]
	if !ok {
		var zero T
		return zero, repository.ErrNotFound
	}
	return doc, nil
}

func (c *collection[T]) GetMany(_ context.Context, ids []bson.ObjectID) ([]T, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	docs := make([]T, 0, len(ids))
	for _, id := range ids {
		if doc, ok := c.docs[id]; ok {
			docs = append(docs, doc)
		}
	}
	return c.sorted(docs), nil
}

func (c *collection[T]) Create(_ context.Context, doc T) (T, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Generam ObjectID ca MongoDB, ca URL-urile sa ramana la fel dupa migrare.
	c.spec.setID(&doc, bson.NewObjectID())
	if err := c.checkUnique(doc); err != nil {
		var zero T
		return zero, err
	}
	c.docs[c.spec.idOf(doc)] = doc
	return doc, nil
}

func (c *collection[T]) Replace(_ context.Context, doc T) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.checkUnique(doc); err != nil {
		return false, err
	}
	id := c.spec.idOf(doc)
	prev, existed := c.docs[id]
	if existed && c.spec.keepLinks != nil {
		c.spec.keepLinks(&doc, prev)
	}
	c.docs[id] = doc
	return !existed, nil
}

func (c *collection[T]) Delete(_ context.Context, id bson.ObjectID) (T, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	doc, ok := c.docs[id]
	if !ok {
		var zero T
		return zero, repository.ErrNotFound
	}
	delete(c.docs, id)
	return doc, nil
}

// update aplica change pe documentul cu id-ul dat, sub Lock, si spune daca l-a modificat.
// E echivalentul unui UpdateOne din MongoDB pe un singur document.
func (c *collection[T]) update(id bson.ObjectID, change func(*T) bool) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	doc, ok := c.docs[id]
	if !ok {
		return false, repository.ErrNotFound
	}
	if !change(&doc) {
		return false, nil
	}
	c.docs[id] = doc
	return true, nil
}

// updateAll aplica change pe toate documentele (UpdateMany in MongoDB), pentru stergerea in cascada.
func (c *collection[T]) updateAll(change func(*T) bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for id, doc := range c.docs {
		if change(&doc) {
			c.docs[id] = doc
		}
	}
}

// listWhere intoarce o pagina din documentele pentru care match e adevarat, plus numarul lor.
func (c *collection[T]) listWhere(match func(T) bool, offset, limit int) ([]T, int) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var found []T
	for _, doc := range c.docs {
		if match(doc) {
			found = append(found, doc)
		}
	}
	found = c.sorted(found)
	total := len(found)
	start := min(offset, total)
	end := min(start+limit, total)
	return found[start:end], total
}

// find intoarce primul document (in ordinea id-urilor) pentru care match e adevarat.
func (c *collection[T]) find(match func(T) bool) (T, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, doc := range c.sorted(slices.Collect(maps.Values(c.docs))) {
		if match(doc) {
			return doc, nil
		}
	}
	var zero T
	return zero, repository.ErrNotFound
}

// checkUnique cauta un alt document cu aceeasi valoare intr-un camp unic; se apeleaza sub Lock.
// Valorile goale (ex. hostname omis) nu se compara: un camp lipsa nu intra in conflict.
func (c *collection[T]) checkUnique(doc T) error {
	if c.spec.unique == nil {
		return nil
	}
	id := c.spec.idOf(doc)
	for _, field := range c.spec.unique(doc) {
		if field.value == "" {
			continue
		}
		for otherID, other := range c.docs {
			if otherID != id && slices.Contains(c.spec.unique(other), field) {
				return &repository.DuplicateError{Field: field.name}
			}
		}
	}
	return nil
}
