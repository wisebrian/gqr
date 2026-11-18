package payload

import "testing"

func TestWiFi(t *testing.T) {
	tests := []struct {
		name string
		wifi WiFi
		want string
	}{
		{"WPA by default", WiFi{SSID: "Home", Password: "secret"}, "WIFI:T:WPA;S:Home;P:secret;;"},
		{"open network", WiFi{SSID: "Guest"}, "WIFI:T:nopass;S:Guest;;"},
		{"WEP", WiFi{SSID: "Old", Password: "12345", Security: "WEP"}, "WIFI:T:WEP;S:Old;P:12345;;"},
		{"hidden", WiFi{SSID: "Attic", Password: "x", Hidden: true}, "WIFI:T:WPA;S:Attic;P:x;H:true;;"},
		{"escaping", WiFi{SSID: `a;b,c:d\e"f`, Password: `p;w`}, `WIFI:T:WPA;S:a\;b\,c\:d\\e\"f;P:p\;w;;`},
		{"security in any case", WiFi{SSID: "Old", Password: "12345", Security: "wep"}, "WIFI:T:WEP;S:Old;P:12345;;"},
		// An open network leaves the password out, however nopass is spelled.
		{"open network in capitals", WiFi{SSID: "Cafe", Password: "secret", Security: "NOPASS"}, "WIFI:T:nopass;S:Cafe;;"},
		// An unknown type is escaped: it cannot name another network.
		{
			"unknown security",
			WiFi{SSID: "Cafe", Password: "x", Security: "WPA;S:Evil;P:attacker;H:true;"},
			`WIFI:T:WPA\;S\:Evil\;P\:attacker\;H\:true\;;S:Cafe;P:x;;`,
		},
	}
	for _, tt := range tests {
		if got := tt.wifi.String(); got != tt.want {
			t.Errorf("%s:\n got %s\nwant %s", tt.name, got, tt.want)
		}
	}
}

func TestContact(t *testing.T) {
	tests := []struct {
		name    string
		contact Contact
		want    string
	}{
		{
			"all fields",
			Contact{
				FirstName: "John", LastName: "Smith",
				Organization: "Example, Inc.", Title: "Engineer",
				Phone: "+15555550100", Email: "john@example.com", URL: "https://example.com",
			},
			"BEGIN:VCARD\r\nVERSION:3.0\r\nN:Smith;John;;;\r\nFN:John Smith\r\n" +
				"ORG:Example\\, Inc.\r\nTITLE:Engineer\r\nTEL:+15555550100\r\n" +
				"EMAIL:john@example.com\r\nURL:https://example.com\r\nEND:VCARD\r\n",
		},
		{
			"organization only, with special characters",
			Contact{Organization: "Acme;\nInc"},
			"BEGIN:VCARD\r\nVERSION:3.0\r\nN:;;;;\r\nFN:Acme\\;\\nInc\r\n" +
				"ORG:Acme\\;\\nInc\r\nEND:VCARD\r\n",
		},
		{
			// A URL is not escaped, but it cannot break the line.
			"URL with punctuation and a line break",
			Contact{FirstName: "John", URL: "https://example.com/a,b;c\r\nTEL:1"},
			"BEGIN:VCARD\r\nVERSION:3.0\r\nN:;John;;;\r\nFN:John\r\n" +
				"URL:https://example.com/a,b;cTEL:1\r\nEND:VCARD\r\n",
		},
	}
	for _, tt := range tests {
		if got := tt.contact.String(); got != tt.want {
			t.Errorf("%s:\n got %q\nwant %q", tt.name, got, tt.want)
		}
	}
}
