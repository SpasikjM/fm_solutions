#!/bin/bash

echo "Building binary..."
GOOS=linux GOARCH=amd64 go build -o miloshspasikjapp

echo "Rsyncing to server..."
rsync -r www miloshspasikjapp deploy@miloshspasikj.com:/tmp/miloshspasikjapp_tmp/

ssh -t deploy@miloshspasikj.com '
	export NEW_DIR=miloshspasikj.com_$(date +%F_%H:%M:%S) && \
	cd /sites && \
	echo Creating new dir $NEW_DIR && \
	sudo mkdir $NEW_DIR && \
	echo "Copying files.." && \
	sudo cp -r /tmp/miloshspasikjapp_tmp/* $NEW_DIR && \
	echo "Updating symlink.." && \
	sudo ln -snf $NEW_DIR miloshspasikj.com && \
	echo "Updating permissions.." && \
	sudo chown -R www-data:www-data $NEW_DIR && \
	echo "Restarting app.." && \
	sudo systemctl restart miloshspasikjapp'
echo "Complete"