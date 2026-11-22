package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func ps(script string) (string, error) {
	return run("powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script)
}

func number(d Drive) string { return strings.TrimPrefix(d.Path, `\\.\PhysicalDrive`) }

// drives: USB and card-reader disks only, never the system or boot disk.
func drives() ([]Drive, error) {
	out, err := ps(`$d = @(Get-Disk | Where-Object { -not $_.IsSystem -and -not $_.IsBoot -and ('USB','SD','MMC' -contains $_.BusType) } | ` +
		`Select-Object Number,FriendlyName,Size,@{n='Bus';e={"$($_.BusType)"}}); ConvertTo-Json -Compress -InputObject $d`)
	if err != nil { return nil, err }
	var raw []struct{ Number int; FriendlyName string; Size int64; Bus string }
	if s := strings.TrimSpace(out); s != "" { if err := json.Unmarshal([]byte(s), &raw); err != nil { return nil, err } }
	var ds []Drive
	for _, r := range raw {
		ds = append(ds, Drive{Path: fmt.Sprintf(`\\.\PhysicalDrive%d`, r.Number), Name: r.FriendlyName, Size: r.Size, Via: strings.ToLower(r.Bus)})
	}
	return ds, nil
}

// release: take every volume off the disk so Windows lets go of it.
func release(d Drive) error {
	n := number(d)
	_, err := ps(`$d = Get-Disk -Number ` + n + `; if ($d.IsOffline) { Set-Disk -Number ` + n + ` -IsOffline $false }; ` +
		`if ($d.IsReadOnly) { Set-Disk -Number ` + n + ` -IsReadOnly $false }; ` +
		`if ($d.PartitionStyle -ne 'RAW') { Clear-Disk -Number ` + n + ` -RemoveData -RemoveOEM -Confirm:$false }`)
	return err
}

// putOnBoot: a file onto the card's FAT (boot) partition. Not yet run on a real Windows PC.
func putOnBoot(d Drive, name string, data []byte) error {
	n := number(d)
	out, err := ps(`Update-HostStorageCache; Start-Sleep -Seconds 2; ` +
		`$p = Get-Partition -DiskNumber ` + n + ` | Where-Object { (Get-Volume -Partition $_ -ErrorAction SilentlyContinue).FileSystem -like 'FAT*' } | Select-Object -First 1; ` +
		`if (-not $p) { exit 3 }; if (-not $p.DriveLetter) { $p | Add-PartitionAccessPath -AssignDriveLetter; ` +
		`$p = Get-Partition -DiskNumber ` + n + ` -PartitionNumber $p.PartitionNumber }; Write-Output $p.DriveLetter`)
	letter := strings.TrimSpace(out)
	if err != nil || len(letter) != 1 { return errors.New("the card's FAT partition didn't get a drive letter") }
	return os.WriteFile(letter+`:\`+name, data, 0600)
}

func eject(d Drive) { ps(`Update-HostStorageCache`) }

func elevated(self string, args ...string) error {
	if _, err := run("net", "session"); err == nil { _, err := run(self, args...); return err }
	q := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
	list := make([]string, len(args))
	for i, a := range args { list[i] = q(`"` + a + `"`) }
	_, err := ps(`Start-Process -FilePath ` + q(self) + ` -ArgumentList ` + strings.Join(list, ",") + ` -Verb RunAs -WindowStyle Hidden -Wait`)
	return err
}

func openBrowser(url string) { exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start() }

// pickFile: Windows' own "open a file" window. Empty: the person cancelled.
func pickFile() (string, error) {
	out, _ := exec.Command("powershell", "-NoProfile", "-STA", "-Command",
		`Add-Type -AssemblyName System.Windows.Forms; $d = New-Object System.Windows.Forms.OpenFileDialog; `+
			`$d.Title = 'Choose the image'; $d.Filter = 'Images|*.img;*.xz;*.gz|All files|*.*'; `+
			`if ($d.ShowDialog() -eq 'OK') { $d.FileName }`).Output()
	return strings.TrimSpace(string(out)), nil
}
