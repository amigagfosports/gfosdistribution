# GFOS distribution

Base bootable images that take a machine to GFOS: write one to a USB stick, an NVMe drive or an SD card, and
start the machine from it.

A base image is a starting point. It is not kept up to date afterwards: once a machine is GFOS, the GFOS
software store does the last mile and every update after it.

## Machines

| Folder | Machine | Image |
| --- | --- | --- |
| `cm4/` | Raspberry Pi Compute Module 4 | not built yet |
| `cm5/` | Raspberry Pi Compute Module 5 | not built yet |
| `pi4/` | Raspberry Pi 4 | not built yet |
| `pi5/` | Raspberry Pi 5 | not built yet |
| `x86/` | x86-64 appliance | not built yet |

Planned, when we do them: `riscv/` and `ppc/`.

## What is in a machine's folder

- `README.md`: the machine, what the image needs, how to write it, and what is known not to work yet.
- `base.lock`: the base config, the parts in the image and their versions. The image builder installs exactly
  that list.
- `SHA256SUMS`: the checksum of each published image.

The image files themselves are too big for git. Each published image is a file on a release of this repository,
named `gfos-<machine>-<version>.img.xz`, with its checksum in the machine's `SHA256SUMS`.

## What a base image is not

- It holds no AmigaOS files and no Kickstart ROM: you supply your own, as GFOpenUp asks.
- It is not a copy of one of our test machines. Those copies hold things that stay inside the testing team and
  are never published here.

## Changing a base config

A change to a `base.lock` is a pull request to this repository.

MIT, Copyright (c) 2026 The AmiCradle Team, for our own files.
