#!/usr/bin/env bash

echo " Sorting files in the current directory..."

# Create folders if they don't already exist
mkdir -p Images Documents Code Binaries Configs JSONs 3D

# Move files based on their extensions safely
# The 2>/dev/null hides errors if there are no files of that type
mv *.jpg *.png *.gif Images/ 2>/dev/null
mv *.pdf *.txt *.docx Documents/ 2>/dev/null
mv *.sh *.py *.html Code/ 2>/dev/null
mv *.bin *.exe Binaries/ 2>/dev/null
mv *.cfg *.conf Configs/ 2>/dev/null
mv *.json JSONs/ 2>/dev/null
mv *.stl *.obj *.3mf *.FCStd 3D/ 2>/dev/null

echo " All sorted!"
