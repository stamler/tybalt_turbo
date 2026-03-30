package hooks

import "testing"

func TestNormalizePendingFileName(t *testing.T) {
	t.Run("replaces hash fragments", func(t *testing.T) {
		got := storageUnsafeFileNameChars.Replace("179_mcidjha551_abcd1234.qsimInvoice#1885.pdf")
		want := "179_mcidjha551_abcd1234.qsimInvoice_1885.pdf"
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("replaces query characters", func(t *testing.T) {
		got := storageUnsafeFileNameChars.Replace("receipt_abcd1234?.pdf")
		want := "receipt_abcd1234_.pdf"
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})
}
