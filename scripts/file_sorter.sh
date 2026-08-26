#!/usr/bin/env bash

echo " Sorting files in the current directory..."

# Create folders if they don't already exist
mkdir -p Images Videos Music Documents Scripts Binaries Configs JSONs 3D

# Move files based on their extensions safely
# The 2>/dev/null hides errors if there are no files of that type
mv *.jpg *.png *.gif Images/ 2>/dev/null
mv *.mp4 *.avi *.mov Videos/ 2>/dev/null
mv *.mp3 *.wav *.ogg Music/ 2>/dev/null
mv *.pdf *.txt *.docx Documents/ 2>/dev/null
mv *.sh *.bash Scripts/ 2>/dev/null
mv *.bin *.exe Binaries/ 2>/dev/null
mv *.cfg *.conf Configs/ 2>/dev/null
mv *.json JSONs/ 2>/dev/null
mv *.stl *.obj *.3mf *.FCStd 3D/ 2>/dev/null

echo " All sorted!"
