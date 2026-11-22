# GFOS Imager

A program for a PC or a Raspberry Pi that puts a GFOS base image on a drive without fuss.

Status: not built yet.

What it does:

1. You pick your machine (CM4, CM5, Pi 4, Pi 5, x86).
2. It fetches that machine's base image from this repository's releases, or takes one you already have.
3. It checks the image against its SHA-256, and says so plainly if it does not match.
4. You pick the drive (USB stick, NVMe, SD card). It shows the drive's name and size and asks once before
   writing, and it never offers the drive the computer itself runs from.
5. It writes the image, reads it back to check it, and tells you when the drive can be taken out.

It runs on Windows, Linux (a Raspberry Pi included) and macOS.
