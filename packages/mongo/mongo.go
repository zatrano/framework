package mongo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Client is a MongoDB client (real driver when URI is mongodb:// or mongodb+srv://,
// otherwise an in-memory store for tests and local demos).
type Client struct {
	mu         sync.Mutex
	URI        string
	databases  map[string]*Database
	real       *mongodriver.Client
	connectErr error
}

// Database holds named collections.
type Database struct {
	mu          sync.Mutex
	Name        string
	collections map[string]*Collection
	real        *mongodriver.Database
	memory      bool
}

// Collection stores documents.
type Collection struct {
	mu     sync.Mutex
	Name   string
	docs   []map[string]any
	real   *mongodriver.Collection
	memory bool
}

// Connect opens a client. Use "memory" (or empty) for the in-process store;
// mongodb:// and mongodb+srv:// URIs use the official driver.
func Connect(uri string) *Client {
	if uri == "" {
		uri = "memory"
	}
	c := &Client{
		URI:       uri,
		databases: make(map[string]*Database),
	}
	if isRealMongoURI(uri) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		client, err := mongodriver.Connect(ctx, options.Client().ApplyURI(uri))
		if err != nil {
			c.connectErr = err
			return c
		}
		c.real = client
	}
	return c
}

func isRealMongoURI(uri string) bool {
	u := strings.ToLower(strings.TrimSpace(uri))
	return strings.HasPrefix(u, "mongodb://") || strings.HasPrefix(u, "mongodb+srv://")
}

// Database returns or creates a database.
func (c *Client) Database(name string) *Database {
	if name == "" {
		name = "zatrano"
	}
	if c != nil && c.real != nil {
		return &Database{
			Name:   name,
			real:   c.real.Database(name),
			memory: false,
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	db, ok := c.databases[name]
	if !ok {
		db = &Database{Name: name, collections: make(map[string]*Collection), memory: true}
		c.databases[name] = db
	}
	return db
}

// Ping reports whether the client is reachable.
func (c *Client) Ping() error {
	if c == nil {
		return fmt.Errorf("mongo: nil client")
	}
	if c.connectErr != nil {
		return c.connectErr
	}
	if c.real != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return c.real.Ping(ctx, nil)
	}
	return nil
}

// Close disconnects a real driver client (no-op for memory mode).
func (c *Client) Close() error {
	if c == nil || c.real == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return c.real.Disconnect(ctx)
}

// Collection returns or creates a collection.
func (d *Database) Collection(name string) *Collection {
	if name == "" {
		name = "items"
	}
	if d != nil && d.real != nil {
		return &Collection{Name: name, real: d.real.Collection(name), memory: false}
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.collections == nil {
		d.collections = make(map[string]*Collection)
	}
	col, ok := d.collections[name]
	if !ok {
		col = &Collection{Name: name, docs: make([]map[string]any, 0), memory: true}
		d.collections[name] = col
	}
	return col
}

// InsertOne inserts a document and returns its _id.
func (c *Collection) InsertOne(doc map[string]any) (string, error) {
	if doc == nil {
		return "", fmt.Errorf("mongo: document is required")
	}
	if c != nil && c.real != nil {
		cloned := cloneDoc(doc)
		id, _ := cloned["_id"].(string)
		if id == "" {
			id = newID()
			cloned["_id"] = id
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, err := c.real.InsertOne(ctx, cloned)
		if err != nil {
			return "", err
		}
		return id, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	cloned := cloneDoc(doc)
	id, _ := cloned["_id"].(string)
	if id == "" {
		id = newID()
		cloned["_id"] = id
	}
	c.docs = append(c.docs, cloned)
	return id, nil
}

// Find returns documents matching an equality filter (empty = all).
func (c *Collection) Find(filter map[string]any) ([]map[string]any, error) {
	clean, hostile := sanitizeEqualityFilter(filter)
	if hostile {
		return []map[string]any{}, nil
	}
	if c != nil && c.real != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cur, err := c.real.Find(ctx, toBSONM(clean))
		if err != nil {
			return nil, err
		}
		defer cur.Close(ctx)
		var raw []bson.M
		if err := cur.All(ctx, &raw); err != nil {
			return nil, err
		}
		out := make([]map[string]any, 0, len(raw))
		for _, doc := range raw {
			out = append(out, normalizeDoc(doc))
		}
		return out, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]map[string]any, 0)
	for _, doc := range c.docs {
		if match(doc, clean) {
			out = append(out, cloneDoc(doc))
		}
	}
	return out, nil
}

// FindOne returns the first matching document.
func (c *Collection) FindOne(filter map[string]any) (map[string]any, error) {
	clean, hostile := sanitizeEqualityFilter(filter)
	if hostile {
		return nil, fmt.Errorf("mongo: no documents")
	}
	if c != nil && c.real != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		var raw bson.M
		err := c.real.FindOne(ctx, toBSONM(clean)).Decode(&raw)
		if err != nil {
			if err == mongodriver.ErrNoDocuments {
				return nil, fmt.Errorf("mongo: no documents")
			}
			return nil, err
		}
		return normalizeDoc(raw), nil
	}
	docs, err := c.Find(clean)
	if err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return nil, fmt.Errorf("mongo: no documents")
	}
	return docs[0], nil
}

// UpdateOne sets fields on the first matching document.
func (c *Collection) UpdateOne(filter, update map[string]any) (bool, error) {
	clean, hostile := sanitizeEqualityFilter(filter)
	if hostile {
		return false, nil
	}
	if c != nil && c.real != nil {
		set := bson.M{}
		for k, v := range update {
			if k == "_id" {
				continue
			}
			set[k] = v
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		res, err := c.real.UpdateOne(ctx, toBSONM(clean), bson.M{"$set": set})
		if err != nil {
			return false, err
		}
		return res.MatchedCount > 0, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, doc := range c.docs {
		if match(doc, clean) {
			for k, v := range update {
				if k == "_id" {
					continue
				}
				doc[k] = v
			}
			c.docs[i] = doc
			return true, nil
		}
	}
	return false, nil
}

// DeleteOne removes the first matching document.
func (c *Collection) DeleteOne(filter map[string]any) (bool, error) {
	clean, hostile := sanitizeEqualityFilter(filter)
	if hostile {
		return false, nil
	}
	if c != nil && c.real != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		res, err := c.real.DeleteOne(ctx, toBSONM(clean))
		if err != nil {
			return false, err
		}
		return res.DeletedCount > 0, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, doc := range c.docs {
		if match(doc, clean) {
			c.docs = append(c.docs[:i], c.docs[i+1:]...)
			return true, nil
		}
	}
	return false, nil
}

// Count returns document count.
func (c *Collection) Count() int {
	if c != nil && c.real != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		n, err := c.real.CountDocuments(ctx, bson.M{})
		if err != nil {
			return 0
		}
		return int(n)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.docs)
}

// sanitizeEqualityFilter keeps equality-only filters.
// hostile is true when Mongo operators ($ne, $where, …) appear as keys.
func sanitizeEqualityFilter(filter map[string]any) (map[string]any, bool) {
	out := map[string]any{}
	if len(filter) == 0 {
		return out, false
	}
	hostile := false
	for k, v := range filter {
		if strings.HasPrefix(k, "$") {
			hostile = true
			continue
		}
		if nested, ok := v.(map[string]any); ok {
			safe := map[string]any{}
			for nk, nv := range nested {
				if strings.HasPrefix(nk, "$") {
					hostile = true
					continue
				}
				safe[nk] = nv
			}
			if len(safe) > 0 {
				out[k] = safe
			} else if hostile {
				// nested was only operators
				continue
			}
			continue
		}
		out[k] = v
	}
	return out, hostile
}

func toBSONM(filter map[string]any) bson.M {
	out := bson.M{}
	for k, v := range filter {
		out[k] = v
	}
	return out
}

func normalizeDoc(doc bson.M) map[string]any {
	out := make(map[string]any, len(doc))
	for k, v := range doc {
		out[k] = normalizeValue(v)
	}
	return out
}

func normalizeValue(v any) any {
	switch t := v.(type) {
	case primitive.ObjectID:
		return t.Hex()
	case primitive.A:
		out := make([]any, len(t))
		for i, item := range t {
			out[i] = normalizeValue(item)
		}
		return out
	case bson.M:
		return normalizeDoc(t)
	case bson.D:
		m := bson.M{}
		for _, e := range t {
			m[e.Key] = e.Value
		}
		return normalizeDoc(m)
	case map[string]any:
		return cloneDoc(t)
	default:
		return v
	}
}

func match(doc, filter map[string]any) bool {
	if len(filter) == 0 {
		return true
	}
	for k, want := range filter {
		got, ok := doc[k]
		if !ok {
			return false
		}
		if stringify(got) != stringify(want) {
			return false
		}
	}
	return true
}

func stringify(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	default:
		return fmt.Sprint(v)
	}
}

func cloneDoc(doc map[string]any) map[string]any {
	out := make(map[string]any, len(doc))
	for k, v := range doc {
		out[k] = v
	}
	return out
}

// ParseDatabase extracts a database name hint from a mongodb URI.
func ParseDatabase(uri string) string {
	uri = strings.TrimSpace(uri)
	if uri == "" || uri == "memory" {
		return "zatrano"
	}
	if idx := strings.LastIndex(uri, "/"); idx >= 0 && idx+1 < len(uri) {
		rest := uri[idx+1:]
		if q := strings.IndexAny(rest, "?#"); q >= 0 {
			rest = rest[:q]
		}
		if rest != "" {
			return rest
		}
	}
	return "zatrano"
}

func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return hex.EncodeToString(b[:4]) + "-" + hex.EncodeToString(b[4:6]) + "-" + hex.EncodeToString(b[6:8]) + "-" + hex.EncodeToString(b[8:10]) + "-" + hex.EncodeToString(b[10:])
}
