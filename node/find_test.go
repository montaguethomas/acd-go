package node

import "testing"

func TestFindNode(t *testing.T) {
	// tests are [path -> Id]
	tests := map[string]string{
		"/":                   "/",
		"/README.md":          "/README.md",
		"/rEaDme.MD":          "/README.md",
		"//rEaDme.MD":         "/README.md",
		"///REadmE.Md":        "/README.md",
		"/pictuREs":           "/pictures",
		"/pictuREs/":          "/pictures",
		"/pictures/loGO.png":  "/pictures/logo.png",
		"/pictures//loGO.png": "/pictures/logo.png",
	}

	for path, Id := range tests {
		n, err := Mocked.FindNode(path)
		if err != nil {
			t.Fatalf("MockNodeTree.FindNode(%q) error: %s", path, err)
		}
		if want, got := Id, n.Id; want != got {
			t.Errorf("MockNodeTree.FindNode(%q).Id: want %s got %s", path, want, got)
		}
	}
}

func TestFindById(t *testing.T) {
	tests := []string{
		"/",
		"/README.md",
		"/pictures",
		"/pictures/logo.png",
	}

	for _, test := range tests {
		n, err := Mocked.FindById(test)
		if err != nil {
			t.Errorf("MockNodeTree.FindById(%q) error: %s", test, err)
		}
		if want, got := test, n.Id; want != got {
			t.Errorf("MockNodeTree.FindById(%q).Id: want %s got %s", test, want, got)
		}
	}
}
