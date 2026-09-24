package midtrans

import (
	"testing"

	"github.com/tertua/invoiceman/platform/gateway"
)

// SnapMethods maps neutral ids to Snap codes and never lets an unmappable id
// through, so a restricted account can't accidentally widen its list.
func TestSnapMethodsMapsAndDropsUnknown(t *testing.T) {
	got := SnapMethods([]string{gateway.MethodQRIS, "not_a_method", gateway.MethodGopay})
	want := []string{"qris", "gopay"}
	if len(got) != len(want) {
		t.Fatalf("SnapMethods = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("SnapMethods = %v, want %v", got, want)
		}
	}
}

// An empty or fully-unmappable list yields nil so Snap keeps its default set.
func TestSnapMethodsEmptyYieldsNil(t *testing.T) {
	if SnapMethods(nil) != nil || SnapMethods([]string{}) != nil || SnapMethods([]string{"bogus"}) != nil {
		t.Fatal("SnapMethods should be nil when nothing maps")
	}
}
