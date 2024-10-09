#!/bin/sh /etc/rc.common

# Required fields
START=99
STOP=10

# Binary details
BIN_PATH="/root/rautomate-arm64-linux"
NAME="rautomate"

start() {
    echo "Starting $NAME..."
    start-stop-daemon -S -x $BIN_PATH -b
}

stop() {
    echo "Stopping $NAME..."
    start-stop-daemon -K -x $BIN_PATH
}

restart() {
    echo "Restarting $NAME..."
    stop
    sleep 1
    start
}

status() {
    if pidof $NAME > /dev/null; then
        echo "$NAME is running."
    else
        echo "$NAME is not running."
    fi
}
