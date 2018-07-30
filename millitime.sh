#!/bin/bash
ts=$(date +%s%N) ; $@ ; tt=$((($(date +%s%N) - $ts)/1000000)) ; echo "$(echo $tt)ms"

