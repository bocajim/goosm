goosm
=====

## Overview
"goosm" is a simple tool that reads Open Street Map XML files and imports them into MongoDB with proper geo-spatial indicies.

## Important Notes
* Some data in the WAY collection is filtered by this application for my specific needs, you may want to change this behavior.

## Building

### Pre-requisites
* Must have go installed.
* Must have bzr installed
* Must be able to run shell scripts.
* Must have internet access to download dependencies (via HTTPS).

### Building
* Checkout the project from github.com.
* run build.sh (it will download all required dependencies automatically).

### Running

* -f \<osm file to read\>
* -s \<mongo server:port to connect to\> (127.0.0.1:27017 default)
* -db \<name of mongodb database\> (osm default)

### Examples

    goosm -f miami.osm

    goosm -f miami.osm -db foo

    goosm -f miami.osm -s 127.0.0.1:27017
