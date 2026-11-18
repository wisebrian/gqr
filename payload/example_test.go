package payload_test

import (
	"fmt"

	"github.com/wisebrian/gqr/payload"
)

func ExampleWiFi() {
	fmt.Println(payload.WiFi{SSID: "Home", Password: "secret;123"})
	// Output: WIFI:T:WPA;S:Home;P:secret\;123;;
}
