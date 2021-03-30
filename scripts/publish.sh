#!/usr/bin/env bash

tag=${1#"refs/tags/"}
curl https://proxy.golang.org/github.com/ninedraft/gemax/@v/$tag.info