package main

import (
	"errors"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var (
	diskLine = regexp.MustCompile(`(?m)^/dev/(disk\d+) \(external, physical\)`)
	sizeLine = regexp.MustCompile(`Disk Size:.*\((\d+) Bytes\)`)
	nameLine = regexp.MustCompile(`Device / Media Name:\s*(.+)`)
	busLine  = regexp.MustCompile(`Protocol:\s*(.+)`)
)

// drives: external physical disks only, which is never the one the Mac started from.
func drives() ([]Drive, error) {
	out, err := run("diskutil", "list", "external", "physical")
	if err != nil { return nil, err }
	var ds []Drive
	for _, m := range diskLine.FindAllStringSubmatch(out, -1) {
		info, _ := run("diskutil", "info", m[1])
		d := Drive{Path: "/dev/r" + m[1], Name: m[1]}
		if s := sizeLine.FindStringSubmatch(info); s != nil { d.Size, _ = strconv.ParseInt(s[1], 10, 64) }
		if s := nameLine.FindStringSubmatch(info); s != nil { d.Name = strings.TrimSpace(s[1]) }
		if s := busLine.FindStringSubmatch(info); s != nil { d.Via = strings.ToLower(strings.TrimSpace(s[1])) }
		if d.Size > 0 { ds = append(ds, d) }
	}
	return ds, nil
}

func plain(d Drive) string { return strings.Replace(d.Path, "/dev/r", "/dev/", 1) }

func release(d Drive) error { _, err := run("diskutil", "unmountDisk", "force", plain(d)); return err }

var mountLine = regexp.MustCompile(`Mount Point:\s*(/.+)`)

// putOnBoot: a file onto the card's FAT (boot) partition. Not yet run on a real Mac.
func putOnBoot(d Drive, name string, data []byte) error {
	run("diskutil", "mountDisk", plain(d))
	for _, slice := range []string{"s1", "s15", "s2"} {
		info, _ := run("diskutil", "info", plain(d)+slice)
		if !strings.Contains(info, "FAT") && !strings.Contains(info, "MS-DOS") { continue }
		if m := mountLine.FindStringSubmatch(info); m != nil {
			return os.WriteFile(strings.TrimSpace(m[1])+"/"+name, data, 0600)
		}
	}
	return errors.New("the card's FAT partition didn't mount")
}

func eject(d Drive) { run("diskutil", "eject", plain(d)) }

func elevated(self string, args ...string) error {
	if os.Geteuid() == 0 { _, err := run(self, args...); return err }
	q := func(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }
	cmd := q(self); for _, a := range args { cmd += " " + q(a) }
	script := `do shell script "` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(cmd) + `" with prompt "GFOS Imager needs to write to the drive." with administrator privileges`
	_, err := run("osascript", "-e", script)
	return err
}

func openBrowser(url string) { exec.Command("open", url).Start() }
