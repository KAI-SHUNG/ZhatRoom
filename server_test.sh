#!/usr/bin/bash

go build -o bin/server cmd/server/main.go

if [ $? -ne 0 ]; then
	echo "Fail"
	exit 1
fi

./bin/server
