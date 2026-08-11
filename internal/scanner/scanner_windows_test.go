//go:build windows

package scanner

import (
	"testing"

	"github.com/DylonH78/AppAlias/internal/model"
)

func TestDecodeStartAppsHandlesSingleObjectAndArray(t *testing.T) {
	for _, input := range [][]byte{
		[]byte(`{"Name":"Calculator","AppID":"Microsoft.WindowsCalculator_8wekyb3d8bbwe!App"}`),
		[]byte(`[{"Name":"Calculator","AppID":"Microsoft.WindowsCalculator_8wekyb3d8bbwe!App"}]`),
	} {
		apps, err := decodeStartApps(input)
		if err != nil {
			t.Fatal(err)
		}
		if len(apps) != 1 || apps[0].Name != "Calculator" {
			t.Fatalf("unexpected decode result: %#v", apps)
		}
	}
}

func TestCandidateKeyIgnoresDisplayNameButRetainsLaunchIdentity(t *testing.T) {
	first := model.Candidate{Launch: model.LaunchSpec{Kind: model.LaunchExecutable, Target: `C:\Apps\Demo.exe`, Arguments: []string{"--one"}}}
	second := first
	if candidateKey(first) != candidateKey(second) {
		t.Fatal("the same launch identity should deduplicate")
	}
	second.Launch.Arguments = []string{"--two"}
	if candidateKey(first) == candidateKey(second) {
		t.Fatal("different arguments must not deduplicate")
	}
}

func TestSplitCommandLine(t *testing.T) {
	arguments, err := splitCommandLine(`--profile "work profile" --safe-mode`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--profile", "work profile", "--safe-mode"}
	if len(arguments) != len(want) {
		t.Fatalf("arguments = %#v, want %#v", arguments, want)
	}
	for index := range want {
		if arguments[index] != want[index] {
			t.Fatalf("arguments = %#v, want %#v", arguments, want)
		}
	}
}
