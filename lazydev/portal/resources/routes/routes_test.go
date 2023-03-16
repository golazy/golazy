package routes_test

import (
	"portal/apps/portal"
	"testing"

	"golazy.dev/lazyapp/apptest"
)

func BenchmarkRoutes(b *testing.B) {

	expect := apptest.New(b, portal.App).Expect

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {

		expect("GET", "/golazy/routes").Code(200)

	}
}
