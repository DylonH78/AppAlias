package alias

import "testing"

func TestValidate(t *testing.T) {
	cases := []struct {
		name string
		ok   bool
	}{
		{"wechat", true},
		{"微信", true},
		{"CON", false},
		{"app.exe", true},
		{"bad/name", false},
		{"name.", false},
	}
	for _, tc := range cases {
		err := Validate(tc.name)
		if (err == nil) != tc.ok {
			t.Fatalf("Validate(%q) error = %v, want ok=%v", tc.name, err, tc.ok)
		}
	}
}

func TestSuggestionsForChineseDisplayName(t *testing.T) {
	got := Suggestions("微信", `C:\Apps\WeChat.exe`)
	want := map[string]bool{"wechat": true, "微信": true, "wei-xin": true, "wx": true}
	for _, value := range got {
		delete(want, value)
	}
	if len(want) != 0 {
		t.Fatalf("missing suggestions: %#v; got %#v", want, got)
	}
}
