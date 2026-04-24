package sh

import (
	"bytes"
	"encoding/xml"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestUnmarshalJSON(t *testing.T) {
	var a int
	s := NewSession()
	s.ShowCMD = true
	err := s.Command("echo", []string{"1"}).UnmarshalJSON(&a)
	if err != nil {
		t.Error(err)
	}
	if a != 1 {
		t.Errorf("expect a tobe 1, but got %d", a)
	}
}

func TestUnmarshalXML(t *testing.T) {
	s := NewSession()
	xmlSample := `<?xml version="1.0" encoding="utf-8"?>
<server version="1" />`
	type server struct {
		XMLName xml.Name `xml:"server"`
		Version string   `xml:"version,attr"`
	}
	data := &server{}
	s.Command("echo", xmlSample).UnmarshalXML(data)
	if data.Version != "1" {
		t.Error(data)
	}
}

func TestPipe(t *testing.T) {
	s := NewSession()
	s.ShowCMD = true
	s.Call("echo", "hello")
	err := s.Command("echo", "hi").Command("cat", "-n").Start()
	if err != nil {
		t.Error(err)
	}
	err = s.Wait()
	if err != nil {
		t.Error(err)
	}
	out, err := s.Command("echo", []string{"hello"}).Output()
	if err != nil {
		t.Error(err)
	}
	if string(out) != "hello\n" {
		t.Error("capture wrong output:", out)
	}
	s.Command("echo", []string{"hello\tworld"}).Command("cut", []string{"-f2"}).Run()
}

func TestPipeCommand(t *testing.T) {
	c1 := exec.Command("echo", "good")
	rd, wr := io.Pipe()
	c1.Stdout = wr
	c2 := exec.Command("cat", "-n")
	c2.Stdout = os.Stdout
	c2.Stdin = rd
	c1.Start()
	c2.Start()

	c1.Wait()
	wc, ok := c1.Stdout.(io.WriteCloser)
	if ok {
		wc.Close()
	}
	c2.Wait()
}

func TestPipeInput(t *testing.T) {
	s := NewSession()
	s.ShowCMD = true
	s.SetInput("first line\nsecond line\n")
	out, err := s.Command("grep", "second").Output()
	if err != nil {
		t.Error(err)
	}
	if string(out) != "second line\n" {
		t.Error("capture wrong output:", out)
	}
}

func TestTimeout(t *testing.T) {
	s := NewSession()
	err := s.Command("sleep", "2").Start()
	if err != nil {
		t.Fatal(err)
	}
	err = s.WaitTimeout(time.Second)
	if err != ErrExecTimeout {
		t.Fatal(err)
	}
}

func TestSetTimeout(t *testing.T) {
	s := NewSession()
	s.SetTimeout(time.Second)
	defer s.SetTimeout(0)
	err := s.Command("sleep", "2").Run()
	if err != ErrExecTimeout {
		t.Fatal(err)
	}
}

func TestCombinedOutput(t *testing.T) {
	s := NewSession()
	bytes, err := s.Command("sh", "-c", "echo stderr >&2 ; echo stdout").CombinedOutput()
	if err != nil {
		t.Error(err)
	}
	stringOutput := string(bytes)
	if !(strings.Contains(stringOutput, "stdout") && strings.Contains(stringOutput, "stderr")) {
		t.Errorf("expect output from both output streams, got '%s'", strings.TrimSpace(stringOutput))
	}
}

func TestPipeFail(t *testing.T) {
	sh := NewSession()
	sh.PipeFail = true
	sh.PipeStdErrors = true
	sh.Command("cat", "unknown-file")
	sh.Command("echo")
	if _, err := sh.Output(); err == nil {
		t.Error("expected error")
	}
}

func TestLeafCommand(t *testing.T) {
	s := NewSession()
	s.ShowCMD = true
	s.Command("echo", "hello world")
	s.LeafCommand("xargs")
	s.LeafCommand("xargs")
	out, err := s.Output()
	if err != nil {
		t.Error(err)
	}

	if string(out) != "hello world\nhello world\n" {
		t.Error("capture wrong output:", string(out))
	}
}

func TestLeafCommandSeparateEnvs(t *testing.T) {
	s := NewSession()
	s.ShowCMD = true
	var args1 []interface{}
	var args2 []interface{}

	mp := make(map[string]string)
	mp["COMPANY_NAME"] = "APPSCODE"
	args1 = append(args1, "COMPANY_NAME")
	args1 = append(args1, mp)
	s.LeafCommand("printenv", args1...)

	mp["COMPANY_NAME"] = "GOOGLE"
	args2 = append(args2, "COMPANY_NAME")
	args2 = append(args2, mp)
	s.LeafCommand("printenv", args2...)

	out, err := s.Output()
	if err != nil {
		t.Error(err)
	}
	if string(out) != "APPSCODE\nGOOGLE\n" {
		t.Error("capture wrong output:", string(out))
	}
}

func TestCurrentOutput(t *testing.T) {
	s := NewSession()
	s.ShowCMD = true
	s.Command("seq", "1", "100")
	s.LeafCommand("xargs")
	s.LeafCommand("xargs")

	// Enable output buffering and start the pipeline
	s.enableOutputBuffer = true
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}

	// Poll CurrentOutput while the pipeline runs
	done := make(chan error, 1)
	go func() {
		done <- s.Wait()
	}()

	// Poll a few times before the pipeline finishes
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

loop:
	for {
		select {
		case <-ticker.C:
			// Safe to call concurrently with Write
			_, _ = s.CurrentOutput(0)
			_, _ = s.CurrentOutput(1)
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
			break loop
		}
	}

	// After completion, both leaf outputs should be non-empty
	out0, err := s.CurrentOutput(0)
	if err != nil {
		t.Fatal(err)
	}
	out1, err := s.CurrentOutput(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(out0) == 0 {
		t.Error("expected non-empty output for leaf 0")
	}
	if len(out1) == 0 {
		t.Error("expected non-empty output for leaf 1")
	}

	// Both leaf commands get the same parent output, so they should match
	if string(out0) != string(out1) {
		t.Errorf("expected identical outputs, got:\nleaf0: %q\nleaf1: %q", string(out0), string(out1))
	}

	// Out-of-range index should return an error
	_, err = s.CurrentOutput(5)
	if err == nil {
		t.Error("expected error for out-of-range index")
	}
}

func TestCurrentOutputForCommandChain(t *testing.T) {
	s := NewSession()
	s.Command("sh", "-c", "for i in $(seq 1 20); do echo $i; sleep 0.01; done").Command("cat")

	s.enableOutputBuffer = true
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		done <- s.Wait()
	}()

	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.After(3 * time.Second)

	sawProgress := false
loop:
	for {
		select {
		case <-ticker.C:
			out, err := s.CurrentOutput(0)
			if err != nil {
				t.Fatal(err)
			}
			if len(out) > 0 {
				sawProgress = true
			}
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
			break loop
		case <-timeout:
			t.Fatal("pipeline did not finish in time")
		}
	}

	out, err := s.CurrentOutput(0)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "1\n") || !strings.Contains(string(out), "20\n") {
		t.Fatalf("unexpected output: %q", string(out))
	}
	if !sawProgress {
		t.Fatal("expected to observe non-empty in-flight output")
	}

	if _, err := s.CurrentOutput(1); err == nil {
		t.Fatal("expected out-of-range error for command chain")
	}
}

func TestCurrentLeafOutputCompatibility(t *testing.T) {
	s := NewSession()
	s.Command("echo", "hello")
	s.enableOutputBuffer = true
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	if err := s.Wait(); err != nil {
		t.Fatal(err)
	}

	out1, err := s.CurrentOutput(0)
	if err != nil {
		t.Fatal(err)
	}
	out2, err := s.CurrentLeafOutput(0)
	if err != nil {
		t.Fatal(err)
	}
	if string(out1) != string(out2) {
		t.Fatalf("compat mismatch: CurrentOutput=%q CurrentLeafOutput=%q", string(out1), string(out2))
	}
}

func TestOutputThenRunKeepsRunWorking(t *testing.T) {
	s := NewSession()
	s.Command("echo", "hello")

	out, err := s.Output()
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "hello\n" {
		t.Fatalf("unexpected output(): %q", string(out))
	}

	// Rebuild command list before next run because exec.Cmd cannot be started twice.
	s.Command("echo", "hello")
	var runOut bytes.Buffer
	s.Stdout = &runOut
	if err := s.Run(); err != nil {
		t.Fatal(err)
	}
	if got := runOut.String(); got != "hello\n" {
		t.Fatalf("run output mismatch, got %q", got)
	}
}

func TestOutputTwiceNoDuplicate(t *testing.T) {
	s := NewSession()
	s.Command("echo", "hello")

	out1, err := s.Output()
	if err != nil {
		t.Fatal(err)
	}
	s.Command("echo", "hello")
	out2, err := s.Output()
	if err != nil {
		t.Fatal(err)
	}

	if string(out1) != "hello\n" {
		t.Fatalf("first output mismatch, got %q", string(out1))
	}
	if string(out2) != "hello\n" {
		t.Fatalf("second output mismatch, got %q", string(out2))
	}
}
