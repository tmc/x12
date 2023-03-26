#!/bin/bash
# fetch-examples.sh crawls the x12.org examples and fetches them to use as test data
for url in $(cat urls.txt); do
	c1=$(curl -s https://x12.org$url |pup 'a attr{href}' | grep $url/example)
	for eurl in $c1; do
		for eeurl in $(curl -s https://x12.org$eurl |pup '.examples-list-item a attr{href}'); do
			dest=$(basename $url)-$(basename $eeurl).edi
			curl -s https://x12.org$eeurl | pup -p 'p.data json{}' |jq -r '.[].text'> $dest
		done
	done
done
