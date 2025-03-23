#!/usr/bin/env bash

source "$(dirname "$(dirname "$(dirname "$0")")")/lib/common.sh"

threshold="$1"
[ -z "$threshold" ] && threshold=80

log_info "Verificando uso de disco..."

df -h | awk -v threshold="$threshold" '
NR>1 {
    gsub(/%/,"",$5)
    if($5 > threshold) {
        print "ALERTA: Partição " $6 " está com " $5 "% de uso!"
    }
}' 