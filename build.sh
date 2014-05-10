#!/bin/sh

os=`uname`
echo "Detected OS: $os"

case $os in
	"Linux")
		export GOPATH=`pwd`
		export GOBIN=`pwd`/bin
		;;
	"Darwin")
		export GOPATH=`pwd`
		export GOBIN=`pwd`/bin
		;;
	"MINGW32_NT-6.1")
		export GOPATH=`pwd`
		export GOBIN=`pwd`/bin
		;;
	*)
		echo "ERROR: Unknown OS."
		;;
esac

if [ ! -d "src/github.com/cheggaaa/pb" ]; then
	echo "Downloading: github.com/cheggaaa/pb"
	go get github.com/cheggaaa/pb
fi
if [ ! -d "src/github.com/olekukonko/ts" ]; then
	echo "Downloading: github.com/olekukonko/ts"
	go get github.com/olekukonko/ts
fi
if [ ! -d "src/labix.org/v2/mgo" ]; then
	echo "Downloading: labix.org/v2/mgo"
	go get labix.org/v2/mgo
fi

function build {
	echo "Building: dwopen"
	if [ -d "pkg" ]; then
    	rm -Rf pkg/*
	fi
	go install src/goosm.go
}

case $1 in
	"build")
		build
		;;
	*)
		build
		;;
esac

echo "Done"
