# GFOS Imager

> **Test build. Not for release.** It erases drives, and it has not yet been proven on a real card on any
> system. Until it has, it refuses to write to a real drive unless started with `-testing`.

Writes a GFOS base image to a USB drive, NVMe or SD card, on Windows, macOS, Linux and Raspberry Pi OS.

1. Start it. Its window opens in your browser (the page is served only to your own computer).
2. Choose your machine and the image. Or give it an image file you already have (`.img`, `.img.xz` or `.img.gz`).
3. Choose the drive. Only removable drives are listed, never the one your computer runs from.
4. If you like, give the card your Wi-Fi name and password, a name, and an SSH password. They go onto the card's boot
   partition as `amicradle-network.txt`, which the card reads at its first start and then takes the passwords out of.
5. Tick the box and press **Write the image**. Your system asks you to allow it.

It downloads the image, checks it against its published checksum, writes it, then reads the drive back and
compares. If any check fails it says so and the drive shouldn't be used.

The list of images comes from `images/index.json` in this repository.

## From a terminal

    gfos-imager -list                              the drives it could write to
    sudo gfos-imager -write base.img.xz -to /dev/sdX
    gfos-imager -catalogue my-list.json            use another list of images

## Building

Go 1.22 or newer (one pure-Go module, for `.xz`):

    ./build.sh        makes dist/ with a program for each system

## State

Version 0.1. Written and tested on Linux against a stand-in drive. The Windows and macOS drive handling has
been compiled but not yet run on a real machine, and no build has yet written a real card.

Each system is switched on in `main.go` (`Proven`) only after a build has written a real card on it and that
card has started a machine. Testers start it with `-testing`, on a drive they can afford to lose.
