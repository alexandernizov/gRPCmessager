#!/bin/sh
#wait-for-redis.sh


set -e

host="$1"
shift
cmd="$@"

echo "Start waiting for Redis fully start."
echo "Try ping Redis... "
PONG=`redis-cli -h $host -p 6379 ping | grep PONG`
while [ -z "$PONG" ]; do
    sleep 1
    echo "Retry Redis ping... "
    PONG=`redis-cli -h $host -p 6379 ping | grep PONG`
done
echo "Redis at host fully started."

exec $cmd