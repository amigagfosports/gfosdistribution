# GFOS distribution

The released GFOS: the base images that take a machine to GFOS, and the released software and firmware that go
on it. The GFOS software store looks here.

## What is here

| Folder | Holds |
| --- | --- |
| `images/` | Base bootable images, one folder per machine: write one to a USB stick, an NVMe drive or an SD card and start the machine from it |
| `software/` | Released software: what the store offers, with its catalogue |
| `firmware/` | Released firmware |
| `imager/` | The GFOS Imager: a program for a PC or a Pi that picks the right base image, checks it and writes it to a drive, without fuss |

Only released things are here. Work in progress stays in each project's own repository.

## Base images

| Folder | Machine | Image |
| --- | --- | --- |
| `images/cm4/` | Raspberry Pi Compute Module 4 | not built yet |
| `images/cm5/` | Raspberry Pi Compute Module 5 | not built yet |
| `images/pi4/` | Raspberry Pi 4 | not built yet |
| `images/pi5/` | Raspberry Pi 5 | not built yet |
| `images/x86/` | x86-64 | not built yet |

Planned, when we do them: `images/riscv/` and `images/ppc/`.

A base image is a starting point. It is not kept up to date afterwards: once a machine is GFOS, the store does
the last mile and every update after it.

Each machine's folder has a `README.md` (what the image needs and how to write it), a `base.lock` (the base
config: the parts in the image and their versions) and, once an image is published, its `SHA256SUMS`.

## Where the files are

Images, archives and firmware files are too big for git. Each is a file on a release of this repository, with its
SHA-256 beside its entry here, so anything fetched can be checked.

## What is not here

No AmigaOS files and no Kickstart ROM: you supply your own, as GFOpenUp asks.

## Changing what is released

A change here is a pull request.

MIT, Copyright (c) 2026 The AmiCradle Team, for our own files. Third-party parts keep their own licences.
