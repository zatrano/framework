package sitemap_test

import (
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/sitemap"
)

func TestSitemapXML(t *testing.T) {
	b := sitemap.New("http://localhost:8080")
	b.Add("/").Add("/documentation")
	xml := b.XML()
	if !strings.Contains(xml, "<loc>http://localhost:8080/</loc>") {
		t.Fatal(xml)
	}
	robots := b.Robots()
	if !strings.Contains(robots, "Sitemap: http://localhost:8080/sitemap.xml") {
		t.Fatal(robots)
	}
}
