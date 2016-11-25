# uvgTorrent-golang

uvgTorrent-golang is an educational torrent streaming client built with golang. It isn't intended for regular use, but demonstrates using a magnet link to connect to peers via the udp tracker protocol, downloading torrent metadata from those peers and then streaming a selected (**__legally shared__**) video file by downloading the file in sequential order and playing it through vnc. 

Besides demonstrating the basics of the torrent protocol uvgTorrent also shows off just how suited golang is for high concurrency, network heavy programs. Golang's built in concurrency support makes it easy to handle communication with each peer in a seperate goroutine, using channels to orchastrate loading metadata, dividing pieces between the peers for download, downloading the pieces, and saving them to the harddrive.

I intend to use this project as the foundation of a tutorial serious, so if you have any recomendations or feedback, give me a shout at smnbursten@gmail.com

## branches

the master branch includes a simple ui developed using termui (https://github.com/gizak/termui). once the file list loads use the up and down keys to hilight the file you want to watch, and enter to begin downloading it.

if you want to take a look at the simplest working version of the code take a look at branch 'barebones'. when the file list loads you'll see an id next to each file. just enter the id into the console and press enter to start downloading it.

## requirements

- golang 1.6.3
- vlc (if you want to open vlc to stream the video file you select for download)

## usage

```bash
#!/bin/bash

export GOPATH=/absolute/path/to/uvgTorrent-golang/
go run "magnet:magneturlgoeshere"
```

## torrent protocol background

If you want to read up on the torrent protocol start here:
    - https://wiki.theory.org/BitTorrentSpecification
    - http://bittorrent.org/beps/bep_0003.html

## technical description