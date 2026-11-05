#!/bin/bash

set -euo pipefail

# Requires a terminal supporting Kitty graphics. Ttyphoon does not yet support it.
# Through tmux, passthrough must be enabled: tmux set -g allow-passthrough on

payload=$(
    for ((y = 0; y < 16; y++)); do
        for ((x = 0; x < 16; x++)); do
            if (((x / 4 + y / 4) % 2 == 0)); then
                printf '\377\120\100'
            else
                printf '\100\240\377'
            fi
        done
    done | base64 | tr -d '\r\n'
)

printf 'Kitty graphics: orange and blue checkerboard (24 columns by 12 rows)\n'

# Direct RGB transmission: 16x16 pixels, displayed in a 24x12-cell placement.
# q=2 suppresses protocol replies so they cannot become shell input.
sequence=$(printf '\033_Ga=T,t=d,f=24,s=16,v=16,c=24,r=12,q=2;%s\033\\' "$payload")
if [[ -n "${TMUX:-}" ]]; then
    printf '\033Ptmux;%s\033\\' "${sequence//$'\033'/$'\033\033'}"
else
    printf '%s' "$sequence"
fi

printf '\n'
