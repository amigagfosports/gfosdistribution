package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type blk struct {
	Name, Path, Model, Tran, Type string
	Size                          json.Number
	RM, Hotplug                   any
	Mountpoint                    *string
	Children                      []blk
}

func yes(v any) bool { return v == true || v == "1" }

func (b blk) holdsSystem() bool {
	if b.Mountpoint != nil {
		m := *b.Mountpoint
		if m == "/" || strings.HasPrefix(m, "/boot") || m == "/usr" || m == "/home" || m == "[SWAP]" { return true }
	}
	for _, c := range b.Children { if c.holdsSystem() { return true } }
	return false
}

// drives: removable ones only (USB, card readers), never the disk this computer is running from.
func drives() ([]Drive, error) {
	out, err := run("lsblk", "-J", "-b", "-o", "NAME,PATH,SIZE,MODEL,TRAN,RM,HOTPLUG,TYPE,MOUNTPOINT")
	if err != nil { return nil, err }
	var t struct{ Blockdevices []blk }
	if err := json.Unmarshal([]byte(out), &t); err != nil { return nil, err }
	var ds []Drive
	for _, b := range t.Blockdevices {
		size, _ := b.Size.Int64()
		card := strings.HasPrefix(b.Name, "mmcblk") && !strings.Contains(b.Name, "boot")
		if b.Type != "disk" || size == 0 || b.holdsSystem() || !(yes(b.RM) || yes(b.Hotplug) || b.Tran == "usb" || card) { continue }
		via := b.Tran; if card { via = "sd" }
		name := strings.TrimSpace(b.Model); if name == "" { name = b.Name }
		ds = append(ds, Drive{Path: b.Path, Name: name, Size: size, Via: via})
	}
	return ds, nil
}

func release(d Drive) error {
	out, _ := run("lsblk", "-ln", "-o", "PATH,MOUNTPOINT", d.Path)
	for _, l := range strings.Split(out, "\n") {
		if f := strings.Fields(l); len(f) == 2 { if _, err := run("umount", f[0]); err != nil { return err } }
	}
	return nil
}

func eject(d Drive) { run("sync") }

// putOnBoot: a file onto the card's FAT (boot) partition.
func putOnBoot(d Drive, name string, data []byte) error {
	run("partprobe", d.Path); run("udevadm", "settle")
	out, _ := run("lsblk", "-ln", "-o", "PATH,FSTYPE", d.Path)
	part := ""
	for _, l := range strings.Split(out, "\n") {
		if f := strings.Fields(l); len(f) == 2 && f[1] == "vfat" && part == "" { part = f[0] }
	}
	if part == "" { return errors.New("the card has no FAT partition to put them on") }
	dir, err := os.MkdirTemp("", "gfos-imager-boot"); if err != nil { return err }
	defer os.Remove(dir)
	if _, err := run("mount", "-t", "vfat", part, dir); err != nil { return err }
	werr := os.WriteFile(dir+"/"+name, data, 0600)
	run("sync")
	if _, err := run("umount", dir); err != nil && werr == nil { werr = err }
	return werr
}

func elevated(self string, args ...string) error {
	if os.Geteuid() == 0 { _, err := run(self, args...); return err }
	if _, err := exec.LookPath("pkexec"); err != nil {
		return errors.New("writing needs administrator rights: start GFOS Imager with sudo")
	}
	if _, err := run("pkexec", append([]string{self}, args...)...); err != nil { return fmt.Errorf("%v", err) }
	return nil
}

func openBrowser(url string) { exec.Command("xdg-open", url).Start() }
