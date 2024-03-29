#!/bin/sh

GOVERSION=${1-"1.22.1"}

git clone https://github.com/syndbg/goenv.git /usr/local/goenv

export GOENV_ROOT=/usr/local/goenv
/usr/local/goenv/bin/goenv install ${GOVERSION}
/usr/local/goenv/bin/goenv global ${GOVERSION}
