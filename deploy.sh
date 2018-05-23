#!/bin/sh
IMAGE=k8r.eu/justjanne/statsbot
TAGS=$(git describe --always --tags HEAD)
DEPLOYMENT=statsbot
POD=statsbot

kubectl -n statsbot set image deployment/$DEPLOYMENT $POD=$IMAGE:$TAGS