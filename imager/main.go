// GFOS Imager: pick a GFOS base image for your machine, check it, and write it to a USB drive, NVMe or SD card.
//
// It runs as you and shows its window in your browser (a page served only to this computer). It lists the
// published base images, downloads the one you pick and checks it against its published checksum. Only the
// writing needs administrator rights: for that it starts a second copy of itself (-write), which your system
// asks you to allow. That copy writes the image, reads it back and compares, and will only write to a drive
// that is removable and isn't the one this computer started from.
package main

import (
	"bufio"
	"compress/gzip"
	"crypto/rand"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/ulikunitz/xz"
)

const Version = "0.1"

// Proven: set true for a system only once a build has written a real card on it and that card has started a
// machine. Until then this program refuses to write to a real drive on that system unless it's started with
// -testing, which is for the people proving it.
var Proven = map[string]bool{"linux": false, "windows": false, "darwin": false}
var testing, large bool

// Large: a drive bigger than this is more likely somebody's backup disk than a card to write, so it's kept out
// of the list, and refused, unless asked for (the "Show large drives" tick, or -large).
const Large = 256e9
const DefaultCatalogue = "https://raw.githubusercontent.com/amigagfosports/gfosdistribution/main/images/index.json"

//go:embed ui.html
var page []byte

type Image struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Machine  string `json:"machine"`
	File     string `json:"file"`
	Size     int64  `json:"size"`
	SHA256   string `json:"sha256"`
	URL      string `json:"url"`
	Notes    string `json:"notes,omitempty"`
	Released string `json:"released,omitempty"`
}

type Drive struct {
	Path  string `json:"path"`  // what gets opened for writing
	Name  string `json:"name"`  // what the person sees
	Size  int64  `json:"size"`
	Via   string `json:"via"`   // usb, sd...
}

// Net: what the card should be told about its network, put on its boot partition as amicradle-network.txt.
// The card reads that file at its first start, applies it, and takes the passwords out of it.
type Net struct {
	Hostname    string `json:"hostname"`
	Country     string `json:"country"`
	SSID        string `json:"ssid"`
	WifiPass    string `json:"wifiPassword"`
	SSHPass     string `json:"sshPassword"`
}

func (n *Net) empty() bool { return n == nil || n.Hostname+n.SSID+n.WifiPass+n.SSHPass == "" }

func (n *Net) text() (string, error) {
	for _, v := range []string{n.Hostname, n.Country, n.SSID, n.WifiPass, n.SSHPass} {
		if strings.ContainsAny(v, "\r\n") || v != strings.TrimSpace(v) { return "", errors.New("A setting can't start or end with a space, or run over a line.") }
	}
	var b strings.Builder
	b.WriteString("# Written by GFOS Imager. The card applies this at its first start and takes the passwords out.\n")
	if n.Hostname != "" { b.WriteString("hostname = " + n.Hostname + "\n") }
	if n.Country != "" { b.WriteString("country = " + strings.ToUpper(n.Country) + "\n") }
	if n.SSID != "" {
		if n.WifiPass != "" && (len(n.WifiPass) < 8 || len(n.WifiPass) > 63) { return "", errors.New("A Wi-Fi password is 8 to 63 characters.") }
		b.WriteString("[wifi]\nssid = " + n.SSID + "\n")
		if n.WifiPass != "" { b.WriteString("password = " + n.WifiPass + "\n") }
	} else if n.WifiPass != "" { return "", errors.New("Give the Wi-Fi network's name with its password.") }
	if n.SSHPass != "" {
		if len(n.SSHPass) < 8 || len(n.SSHPass) > 256 { return "", errors.New("An SSH password is 8 to 256 characters.") }
		b.WriteString("[ssh]\nenabled = yes\npassword = " + n.SSHPass + "\npassword_login = yes\n")
	}
	return b.String(), nil
}

const NetFile = "amicradle-network.txt"

type Progress struct {
	Phase string `json:"phase"` // idle, downloading, checking, waiting, writing, verifying, done, failed
	Done  int64  `json:"done"`
	Total int64  `json:"total"`
	Note  string `json:"note,omitempty"`
}

var (
	mu        sync.Mutex
	prog      = Progress{Phase: "idle"}
	catalogue = DefaultCatalogue
)

func setProg(p Progress) { mu.Lock(); prog = p; mu.Unlock() }

func main() {
	list := flag.Bool("list", false, "print the drives that could be written to, and stop")
	write := flag.String("write", "", "write this image file (.img, .img.xz or .img.gz); with -to")
	to := flag.String("to", "", "the drive to write to")
	report := flag.String("report", "", "write progress to this file (used by the window)")
	network := flag.String("network", "", "a file of network settings to put on the card as "+NetFile+" (it is deleted once read)")
	flag.BoolVar(&testing, "testing", false, "for testers: allow writing to a real drive on a system not yet proven")
	flag.BoolVar(&large, "large", false, "allow writing to a drive bigger than 256 GB")
	flag.StringVar(&catalogue, "catalogue", DefaultCatalogue, "where the list of base images is (a URL or a file)")
	flag.Parse()
	switch {
	case *list:
		ds, err := drives()
		if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
		for _, d := range ds { fmt.Printf("%s\t%.1f GB\t%s\t%s\n", d.Path, float64(d.Size)/1e9, d.Via, d.Name) }
	case *write != "":
		say := func(p Progress) {
			if *report == "" { fmt.Printf("\r%-10s %3d%% %s", p.Phase, pct(p), p.Note); return }
			b, _ := json.Marshal(p); os.WriteFile(*report+".new", b, 0644); os.Rename(*report+".new", *report)
		}
		var settings []byte
		if *network != "" { settings, _ = os.ReadFile(*network); os.Remove(*network) }
		if err := writeImage(*write, *to, settings, say); err != nil {
			say(Progress{Phase: "failed", Note: err.Error()}); fmt.Fprintln(os.Stderr, "\n"+err.Error()); os.Exit(1)
		}
		say(Progress{Phase: "done", Note: "Written and checked."}); fmt.Println()
	default:
		window()
	}
}

func pct(p Progress) int64 { if p.Total <= 0 { return 0 }; return p.Done * 100 / p.Total }

// ---------------------------------------------------------------- the window (a page for this computer only)
func window() {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
	kb := make([]byte, 16); rand.Read(kb); key := hex.EncodeToString(kb)
	mux := http.NewServeMux()
	guard := func(f http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("k") != key { http.Error(w, "no", 403); return }
			f(w, r)
		}
	}
	js := func(w http.ResponseWriter, v any, err error) {
		w.Header().Set("Content-Type", "application/json")
		if err != nil { w.WriteHeader(400); v = map[string]string{"error": err.Error()} }
		json.NewEncoder(w).Encode(v)
	}
	mux.HandleFunc("/", guard(func(w http.ResponseWriter, r *http.Request) { w.Header().Set("Content-Type", "text/html; charset=utf-8"); w.Write(page) }))
	mux.HandleFunc("/api/images", guard(func(w http.ResponseWriter, r *http.Request) { v, err := images(); js(w, v, err) }))
	mux.HandleFunc("/api/drives", guard(func(w http.ResponseWriter, r *http.Request) { v, err := drives(); if v == nil { v = []Drive{} }; js(w, v, err) }))
	mux.HandleFunc("/api/progress", guard(func(w http.ResponseWriter, r *http.Request) { mu.Lock(); p := prog; mu.Unlock(); js(w, p, nil) }))
	mux.HandleFunc("/api/about", guard(func(w http.ResponseWriter, r *http.Request) {
		js(w, map[string]any{"version": Version, "system": runtime.GOOS, "catalogue": catalogue, "proven": Proven[runtime.GOOS], "testing": testing, "large": Large}, nil)
	}))
	mux.HandleFunc("/api/write", guard(func(w http.ResponseWriter, r *http.Request) {
		var q struct{ Image *Image `json:"image"`; File, Drive string; Large bool; Net *Net `json:"net"` }
		if r.Method != "POST" || json.NewDecoder(r.Body).Decode(&q) != nil || q.Drive == "" { js(w, nil, errors.New("Pick an image and a drive.")); return }
		mu.Lock(); busy := prog.Phase != "idle" && prog.Phase != "done" && prog.Phase != "failed"; mu.Unlock()
		if busy { js(w, nil, errors.New("Already working.")); return }
		setProg(Progress{Phase: "checking"})
		go func() {
			if err := job(q.Image, q.File, q.Drive, q.Large, q.Net); err != nil { setProg(Progress{Phase: "failed", Note: err.Error()}) }
		}()
		js(w, map[string]bool{"started": true}, nil)
	}))
	mux.HandleFunc("/api/quit", guard(func(w http.ResponseWriter, r *http.Request) { js(w, map[string]bool{"bye": true}, nil); go func() { time.Sleep(300 * time.Millisecond); os.Exit(0) }() }))
	url := fmt.Sprintf("http://%s/?k=%s", ln.Addr(), key)
	fmt.Println("GFOS Imager", Version, "is open in your browser. If it isn't, go to:\n ", url, "\nClose this window (or press Ctrl+C) to stop it.")
	openBrowser(url)
	http.Serve(ln, mux)
}

func images() ([]Image, error) {
	var raw []byte; var err error
	if strings.HasPrefix(catalogue, "http") {
		c := http.Client{Timeout: 30 * time.Second}
		resp, e := c.Get(catalogue)
		if e != nil { return nil, fmt.Errorf("Couldn't reach the list of images: %v", e) }
		defer resp.Body.Close()
		if resp.StatusCode != 200 { return nil, fmt.Errorf("The list of images isn't there yet (%s).", resp.Status) }
		raw, err = io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	} else { raw, err = os.ReadFile(catalogue) }
	if err != nil { return nil, err }
	var c struct{ Images []Image `json:"images"` }
	if err := json.Unmarshal(raw, &c); err != nil { return nil, fmt.Errorf("The list of images can't be read: %v", err) }
	return c.Images, nil
}

// job: get the image (download and check it, or take the person's own file), then have it written.
func job(img *Image, file, drive string, big bool, net *Net) error {
	var netText string
	if !net.empty() {
		t, err := net.text(); if err != nil { return err }
		netText = t
	}
	if img != nil {
		dir, _ := os.UserCacheDir(); dir = filepath.Join(dir, "gfos-imager"); os.MkdirAll(dir, 0755)
		file = filepath.Join(dir, filepath.Base(img.File))
		if sum, _ := hashFile(file, nil); sum != img.SHA256 {
			if err := download(img, file); err != nil { return err }
			setProg(Progress{Phase: "checking", Total: img.Size})
			sum, err := hashFile(file, func(n int64) { setProg(Progress{Phase: "checking", Done: n, Total: img.Size}) })
			if err != nil { return err }
			if sum != img.SHA256 { os.Remove(file); return errors.New("The download doesn't match its published checksum, so it wasn't used. Try again.") }
		}
	} else if _, err := os.Stat(file); err != nil { return fmt.Errorf("No such file: %s", file) }
	rep := filepath.Join(os.TempDir(), fmt.Sprintf("gfos-imager-%d.json", os.Getpid())); os.Remove(rep)
	defer os.Remove(rep)
	setProg(Progress{Phase: "waiting", Note: "Your system is asking you to allow the write."})
	self, _ := os.Executable()
	done := make(chan error, 1)
	args := []string{"-write", file, "-to", drive, "-report", rep}
	if testing { args = append(args, "-testing") }
	if big { args = append(args, "-large") }
	if netText != "" { // passwords go to the writing copy in a file only this account can read, never on its command line
		nf := filepath.Join(os.TempDir(), fmt.Sprintf("gfos-imager-%d.net", os.Getpid()))
		if err := os.WriteFile(nf, []byte(netText), 0600); err != nil { return err }
		defer os.Remove(nf)
		args = append(args, "-network", nf)
	}
	go func() { done <- elevated(self, args...) }()
	for {
		select {
		case err := <-done:
			var p Progress
			if b, e := os.ReadFile(rep); e == nil && json.Unmarshal(b, &p) == nil && (p.Phase == "done" || p.Phase == "failed") { setProg(p); return nil }
			if err != nil { return fmt.Errorf("The write didn't run (was it allowed?): %v", err) }
			return errors.New("The write stopped without saying why.")
		case <-time.After(400 * time.Millisecond):
			var p Progress
			if b, e := os.ReadFile(rep); e == nil && json.Unmarshal(b, &p) == nil { setProg(p) }
		}
	}
}

func download(img *Image, dest string) error {
	if img.URL == "" { return errors.New("This image has no download address yet.") }
	resp, err := http.Get(img.URL)
	if err != nil { return fmt.Errorf("Couldn't download it: %v", err) }
	defer resp.Body.Close()
	if resp.StatusCode != 200 { return fmt.Errorf("Couldn't download it: %s", resp.Status) }
	f, err := os.Create(dest + ".part")
	if err != nil { return err }
	buf := make([]byte, 1<<20); var n int64
	for {
		k, e := resp.Body.Read(buf)
		if k > 0 { if _, we := f.Write(buf[:k]); we != nil { f.Close(); return we }; n += int64(k); setProg(Progress{Phase: "downloading", Done: n, Total: img.Size}) }
		if e == io.EOF { break }
		if e != nil { f.Close(); return fmt.Errorf("The download broke off: %v", e) }
	}
	if err := f.Close(); err != nil { return err }
	return os.Rename(dest+".part", dest)
}

func hashFile(path string, tick func(int64)) (string, error) {
	f, err := os.Open(path)
	if err != nil { return "", err }
	defer f.Close()
	h := sha256.New(); buf := make([]byte, 4<<20); var n int64
	for {
		k, e := f.Read(buf)
		h.Write(buf[:k]); n += int64(k)
		if tick != nil { tick(n) }
		if e == io.EOF { break }
		if e != nil { return "", e }
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ---------------------------------------------------------------- the write itself (runs with administrator rights)
const block = 4 << 20

func writeImage(src, dev string, settings []byte, say func(Progress)) error {
	var d *Drive
	if strings.HasSuffix(dev, ".testdisk") { // a plain file standing in for a drive, for testing this program
		st, err := os.Stat(dev); if err != nil { return err }
		d = &Drive{Path: dev, Name: "test file", Size: st.Size()}
	} else {
		if !Proven[runtime.GOOS] && !testing {
			return errors.New("This is a test build: writing to a real drive isn't switched on for this system yet. It hasn't been proven on a real card.")
		}
		ds, err := drives(); if err != nil { return err }
		for i := range ds { if ds[i].Path == dev { d = &ds[i] } }
		if d == nil { return fmt.Errorf("%s isn't a removable drive this program may write to.", dev) }
		if float64(d.Size) > Large && !large {
			return fmt.Errorf("%s is %.0f GB: a drive that big is only written to when you ask for large drives.", dev, float64(d.Size)/1e9)
		}
	}
	in, err := os.Open(src); if err != nil { return err }
	defer in.Close()
	st, _ := in.Stat(); total := st.Size()
	var r io.Reader = in
	low := strings.ToLower(src)
	if strings.HasSuffix(low, ".gz") { z, err := gzip.NewReader(in); if err != nil { return fmt.Errorf("That .gz can't be read: %v", err) }; r = z; total = 0
	} else if strings.HasSuffix(low, ".xz") { z, err := xz.NewReader(bufio.NewReaderSize(in, 4<<20)); if err != nil { return fmt.Errorf("That .xz can't be read: %v", err) }; r = z; total = 0
	} else if total > d.Size { return fmt.Errorf("The image (%.1f GB) is bigger than the drive (%.1f GB).", float64(total)/1e9, float64(d.Size)/1e9) }
	if err := release(*d); err != nil { return fmt.Errorf("Couldn't free the drive for writing: %v", err) }
	out, err := os.OpenFile(d.Path, os.O_RDWR, 0); if err != nil { return fmt.Errorf("Couldn't open the drive: %v", err) }
	h := sha256.New(); buf := make([]byte, block); var n int64
	for {
		k, e := io.ReadFull(r, buf)
		if k > 0 {
			h.Write(buf[:k])
			pad := (512 - k%512) % 512 // drives take whole sectors
			for i := k; i < k+pad; i++ { buf[i] = 0 }
			if n+int64(k) > d.Size { out.Close(); return errors.New("The image is bigger than the drive.") }
			if _, we := out.Write(buf[:k+pad]); we != nil { out.Close(); return fmt.Errorf("Writing failed at %d MB: %v", n>>20, we) }
			n += int64(k); say(Progress{Phase: "writing", Done: n, Total: total})
		}
		if e == io.EOF || e == io.ErrUnexpectedEOF { break }
		if e != nil { out.Close(); return fmt.Errorf("Reading the image failed: %v", e) }
	}
	if err := out.Sync(); err != nil { out.Close(); return fmt.Errorf("Finishing the write failed: %v", err) }
	out.Close()
	want := hex.EncodeToString(h.Sum(nil))
	back, err := os.Open(d.Path); if err != nil { return err }
	defer back.Close()
	h2 := sha256.New(); var m int64
	for m < n {
		k, e := io.ReadFull(back, buf)
		if k == 0 { return fmt.Errorf("Reading the drive back failed at %d MB: %v", m>>20, e) }
		if int64(k) > n-m { k = int(n - m) }
		h2.Write(buf[:k]); m += int64(k); say(Progress{Phase: "verifying", Done: m, Total: n})
	}
	if hex.EncodeToString(h2.Sum(nil)) != want { return errors.New("What's on the drive doesn't match the image. The drive may be failing: try another.") }
	if len(settings) > 0 {
		say(Progress{Phase: "settings", Done: n, Total: n})
		var err error
		if strings.HasSuffix(dev, ".testdisk") { err = os.WriteFile(dev+"."+NetFile, settings, 0600) } else { err = putOnBoot(*d, NetFile, settings) }
		if err != nil { return fmt.Errorf("The image is written and checked, but your settings couldn't be put on the card: %v", err) }
	}
	eject(*d)
	return nil
}

func run(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil { return string(out), fmt.Errorf("%s: %v: %s", name, err, strings.TrimSpace(string(out))) }
	return string(out), nil
}
