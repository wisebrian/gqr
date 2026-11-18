// Package payload formats QR code contents that phones act on: joining a
// Wi-Fi network or saving a contact card. Encode the String of a value:
//
//	wifi := payload.WiFi{SSID: "Home", Password: "secret"}
//	code, err := gqr.Encode(wifi.String(), gqr.Medium)
package payload

import "strings"

// WiFi describes a Wi-Fi network. Its String method returns the WIFI:
// payload that phone cameras offer to join.
type WiFi struct {
	SSID     string
	Password string

	// Security is "WPA" (which also covers WPA2 and WPA3), "WEP" or
	// "nopass", in any case. If empty, it is WPA when Password is set and
	// nopass otherwise. Any other value is escaped, like the other fields,
	// so that it cannot add fields of its own.
	Security string

	Hidden bool // the network does not broadcast its SSID
}

// String returns the payload, for example WIFI:T:WPA;S:Home;P:secret;;
func (w WiFi) String() string {
	var security string
	switch {
	case w.Security != "":
		security = securityType(w.Security)
	case w.Password != "":
		security = "WPA"
	default:
		security = "nopass"
	}
	var b strings.Builder
	b.WriteString("WIFI:T:" + security + ";S:" + wifiEscaper.Replace(w.SSID) + ";")
	if security != "nopass" {
		b.WriteString("P:" + wifiEscaper.Replace(w.Password) + ";")
	}
	if w.Hidden {
		b.WriteString("H:true;")
	}
	b.WriteString(";")
	return b.String()
}

// securityType returns a known security type, given in any case, in the
// spelling that phones expect, and escapes any other.
func securityType(s string) string {
	for _, known := range [...]string{"WPA", "WEP", "nopass"} {
		if strings.EqualFold(s, known) {
			return known
		}
	}
	return wifiEscaper.Replace(s)
}

// wifiEscaper escapes the characters that are special in WIFI: payloads.
var wifiEscaper = strings.NewReplacer(`\`, `\\`, `;`, `\;`, `,`, `\,`, `:`, `\:`, `"`, `\"`)

// Contact is a contact card. Its String method returns a vCard 3.0, which
// phone cameras offer to save to the address book.
type Contact struct {
	FirstName, LastName string
	Organization, Title string
	Phone, Email, URL   string
}

// String returns the vCard, leaving out empty fields.
func (c Contact) String() string {
	name := strings.TrimSpace(c.FirstName + " " + c.LastName)
	if name == "" {
		name = c.Organization
	}
	var b strings.Builder
	line := func(key, value string) {
		b.WriteString(key + ":" + value + "\r\n")
	}
	line("BEGIN", "VCARD")
	line("VERSION", "3.0")
	// N and FN are required. N is family;given;additional;prefixes;suffixes.
	line("N", vcardEscaper.Replace(c.LastName)+";"+vcardEscaper.Replace(c.FirstName)+";;;")
	line("FN", vcardEscaper.Replace(name))
	for _, f := range [...]struct{ key, value string }{
		{"ORG", vcardEscaper.Replace(c.Organization)},
		{"TITLE", vcardEscaper.Replace(c.Title)},
		{"TEL", vcardEscaper.Replace(c.Phone)},
		{"EMAIL", vcardEscaper.Replace(c.Email)},
		// A URL is not text but a URI, which vCard does not escape.
		{"URL", urlCleaner.Replace(c.URL)},
	} {
		if f.value != "" {
			line(f.key, f.value)
		}
	}
	line("END", "VCARD")
	return b.String()
}

// vcardEscaper escapes vCard text values as RFC 2426 requires.
var vcardEscaper = strings.NewReplacer(`\`, `\\`, `,`, `\,`, `;`, `\;`, "\r\n", `\n`, "\n", `\n`, "\r", `\n`)

// urlCleaner removes tabs and line breaks from a URL, as browsers do when
// they parse one, so that it cannot end its line of the vCard early.
var urlCleaner = strings.NewReplacer("\t", "", "\n", "", "\r", "")
