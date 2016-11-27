# uvgTorrent-golang

uvgTorrent-golang is an educational torrent streaming client built with golang. It isn't intended for regular use, but demonstrates using a magnet link to connect to peers via the udp tracker protocol, downloading torrent metadata from those peers and then streaming a selected (**__legally shared__**) video file by downloading the file in sequential order and playing it through vlc. 

Besides demonstrating the basics of the torrent protocol uvgTorrent also shows off just how well suited golang is for high concurrency, network heavy programs. Golang's built in concurrency support makes it easy to handle communication with each peer in a seperate goroutine, using channels to orchastrate loading metadata, dividing pieces between the peers for download, downloading the pieces, and saving them to the harddrive.

I intend to use this project as the foundation of a tutorial, so if you have any recomendations or feedback, give me a shout at smnbursten@gmail.com

## branches

The master branch includes a simple ui developed using termui (https://github.com/gizak/termui). Once the file list loads use the up and down keys to hilight the file you want to watch, and enter to begin downloading it. Once it gets above ~10% you can press v to open the file in vlc. To quit press q.

If you want to take a look at the simplest working version of the code take a look at branch 'barebones'. When the file list loads you'll see an id next to each file. Just enter the id into the console and press enter to start downloading it. To quit hit ctrl-c.

## requirements

- golang 1.6.3
- vlc (if you want to open vlc to stream the video file you select for download)

## usage

```bash
#!/bin/bash

export GOPATH=/absolute/path/to/uvgTorrent-golang/
go run uvgTorrent.go "magnet:magneturigoeshere"
```

## torrent protocol background

If you want to read up on the torrent protocol start here:
    - https://wiki.theory.org/BitTorrentSpecification
    - http://bittorrent.org/beps/bep_0003.html
