package mongo_test

import (
	"testing"

	"github.com/zatrano/framework/packages/mongo"
)

func TestMongoCRUD(t *testing.T) {
	client := mongo.Connect("memory")
	if err := client.Ping(); err != nil {
		t.Fatal(err)
	}
	col := client.Database("zatrano").Collection("posts")
	id, err := col.InsertOne(map[string]any{"title": "Hello", "status": "draft"})
	if err != nil || id == "" {
		t.Fatalf("insert id=%s err=%v", id, err)
	}
	doc, err := col.FindOne(map[string]any{"_id": id})
	if err != nil || doc["title"] != "Hello" {
		t.Fatalf("find=%v err=%v", doc, err)
	}
	ok, err := col.UpdateOne(map[string]any{"_id": id}, map[string]any{"status": "published"})
	if err != nil || !ok {
		t.Fatal(err)
	}
	if col.Count() != 1 {
		t.Fatal("count")
	}
	ok, err = col.DeleteOne(map[string]any{"_id": id})
	if err != nil || !ok || col.Count() != 0 {
		t.Fatal("delete failed")
	}
}

func TestMongoOperatorInjectionRejected(t *testing.T) {
	client := mongo.Connect("memory")
	col := client.Database("zatrano").Collection("users")
	_, _ = col.InsertOne(map[string]any{"email": "ada@example.com", "role": "user"})
	_, _ = col.InsertOne(map[string]any{"email": "root@example.com", "role": "admin"})

	// Operator-style nested filter must not match everything.
	docs, err := col.Find(map[string]any{
		"role": map[string]any{"$ne": "user"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 0 {
		t.Fatalf("operator injection should not match docs, got %d", len(docs))
	}

	docs, err = col.Find(map[string]any{"$where": "true"})
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 0 {
		t.Fatalf("$where must not return all docs, got %d", len(docs))
	}
}
