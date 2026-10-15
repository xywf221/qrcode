package qrcode_test

import (
	"fmt"
	"log"
	"os"

	"github.com/dongm/qrcode"
)

// Example shows the basic encoding API. Because "HELLO" is alphanumeric it is
// packed into alphanumeric mode and fits version 1 at level L.
func Example() {
	q, err := qrcode.NewString("HELLO", qrcode.WithLevel(qrcode.Low), qrcode.WithBoostECL(false))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(q.Version(), q.Size(), q.Mask())
	// Output: 1 21 2
}

// ExampleQR_WritePNG writes a PNG to standard output. In real code you would
// pass a file or an http.ResponseWriter.
func ExampleQR_WritePNG() {
	q, err := qrcode.Encode("https://example.com", qrcode.Quartile)
	if err != nil {
		log.Fatal(err)
	}
	if err := q.WritePNG(os.Stdout, qrcode.WithScale(8)); err != nil {
		log.Fatal(err)
	}
}

// ExampleQR_SVG prints an SVG document.
func ExampleQR_SVG() {
	q, err := qrcode.Encode("https://example.com", qrcode.Quartile)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(q.SVG(qrcode.WithScale(8)))
}

// ExampleQR_Terminal prints a QR symbol to a terminal.
func ExampleQR_Terminal() {
	q, err := qrcode.Encode("https://example.com", qrcode.Quartile)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(q.Terminal(qrcode.WithQuietZone(2)))
}
