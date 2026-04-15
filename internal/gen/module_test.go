package gen

import "testing"

func TestNormalizeName(t *testing.T) {
	if normalizeName("Product-Item") != "product_item" {
		t.Fatal(normalizeName("Product-Item"))
	}
	if normalizeName("  ") != "" {
		t.Fatal("expected empty")
	}
}

func TestSnakeToPascal(t *testing.T) {
	if snakeToPascal("product_item") != "ProductItem" {
		t.Fatal(snakeToPascal("product_item"))
	}
	if snakeToPascal("api") != "Api" {
		t.Fatal(snakeToPascal("api"))
	}
}

