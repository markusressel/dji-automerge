# dji-automerge

A utility to automatically detect parts of a video and join them.

## Why

When you record a video with a DJI drone, the video is split into multiple parts. This is done
to prevent data loss in case of a crash. However, this can be annoying when you want to
work with the video in editing.

While there are great tools to join video segments, like f.ex.
gyroflow's [mp4-merge](https://github.com/gyroflow/mp4-merge)
they still require you to manually specify the parts of the video to join, which is a manual
process that needs constant manual intervention.

This tool aims to automate this by automatically detecting the parts of a video
and join them without the need to manually specify the parts.

## How to use

```shell script
> dji-automerge --input /path/to/videos/ --output /path/to/output/
```

