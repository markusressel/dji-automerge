# dji-automerge

A utility to automatically detect parts of a video and join them.

## Why

When you record a video with a DJI drone, the video is split into multiple parts. This is done
to prevent data loss in case of a crash. However, this can be annoying when you want to
work with the video in editing.

While there are great tools to join video segments, like f.ex.
gyroflow's [mp4-merge](https://github.com/gyroflow/mp4-merge)
they still require you to manually specify the parts of the video to join.

This tool aims to automate this by automatically detecting the parts of a video
and join them without the need to manually specify the parts.

## How to use

```shell script
> git clone $repo
> make build
> ./bin/dji-automerge --input /path/to/videos/ [--output /path/to/output/]
```

## How it works

dji-automerge works by comparing the first and last frame of each video segment
and joining them if they are considered "similar enough". The similarity of the
frames is determined using [vitali-fedulov/images4](https://github.com/vitali-fedulov/images4).

# Dependencies

See [go.mod](go.mod)

# License

```
dji-automerge
Copyright (C) 2024  Markus Ressel

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published
by the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
```
