# GFOS Imager

Writes a GFOS base image to a USB drive, NVMe or SD card, on Windows, macOS, Linux and Raspberry Pi OS.

1. Start it. Its window opens in your browser (the page is served only to your own computer).
2. Choose your machine and the image. Or give it an image file you already have (`.img` or `.img.gz`).
3. Choose the drive. Only removable drives are listed, never the one your computer runs from.
4. Tick the box and press **Write the image**. Your system asks you to allow it.

It downloads the image, checks it against its published checksum, writes it, then reads the drive back and
compares. If any check fails it says so and the drive shouldn't be used.

The list of images comes from `images/index.json` in this repository.

## From a terminal

    gfos-imager -list                              the drives it could write to
    sudo gfos-imager -write base.img.gz -to /dev/sdX
    gfos-imager -catalogue my-list.json            use another list of images

## Building

Go 1.22 or newer, nothing else:

    ./build.sh        makes dist/ with a program for each system

## State

Version 0.1. Written and tested on Linux against a stand-in drive. The Windows and macOS drive handling has
been compiled but not yet run on a real machine, and no build has yet written a real card.
