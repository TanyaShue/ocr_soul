package trainer

import "testing"

func TestSoulTypeFromFilename(t *testing.T) {
	name, err := soulTypeFromFilename("御魂-截图-攻击加成_狂骨.png")
	if err != nil {
		t.Fatal(err)
	}
	if name != "狂骨" {
		t.Fatalf("got %q, want 狂骨", name)
	}
}
