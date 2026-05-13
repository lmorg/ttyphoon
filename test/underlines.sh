#!/bin/bash

# Script to print every style of underline using ANSI SGR 4:n sequences

echo "Underline Styles (Kitty/xterm):"
echo ""

# Style 0: None (reset)
printf "Style 0 (none):     \e[4:0mNo underline\e[0m\n"
printf "Style 1 (single):   \e[4:1mSingle underline\e[0m\n"
printf "Style 2 (double):   \e[4:2mDouble underline\e[0m\n"
printf "Style 3 (curly):    \e[4:3mCurly underline\e[0m\n"
printf "Style 4 (dotted):   \e[4:4mDotted underline\e[0m\n"
printf "Style 5 (dashed):   \e[4:5mDashed underline\e[0m\n"

echo ""
echo "Custom underline colours (SGR 58):"
printf "Single + cyan ULC:  \e[38;2;220;220;220;4:1;58:2:0:200:255mText fg grey, underline cyan\e[0m\n"
printf "Curly + magenta:    \e[38;2;220;220;220;4:3;58:2:255:80:180mText fg grey, underline magenta\e[0m\n"
printf "Dotted + orange:    \e[38;2;220;220;220;4:4;58:2:255:170:40mText fg grey, underline orange\e[0m\n"
printf "Dashed + green:     \e[38;2;220;220;220;4:5;58:2:120:220:120mText fg grey, underline green\e[0m\n"
printf "256-colour ULC:     \e[38;5;15;4:1;58:5:196mWhite text, red underline (idx 196)\e[0m\n"

echo ""
echo "Underline colour reset (SGR 59):"
printf "Before reset:       \e[38;2;220;220;220;4:1;58:2:255:0:0mRed underline\e[0m\n"
printf "After SGR 59:       \e[38;2;220;220;220;4:1;58:2:255:0:0m\e[59mBack to text-colour underline\e[0m\n"
