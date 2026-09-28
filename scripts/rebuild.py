#!/usr/bin/env python3

import os
import subprocess
import time
from pathlib import Path
import datetime

directory = Path("_ABSOLUTE_CARROT").resolve()
go_mtimes = {}
index_mtime = None
command = "GOOS=js GOARCH=wasm go build -o ../main.wasm cmd/wasm/main.go"
# command = "GOOS=js GOARCH=wasm tinygo build -o ../main.wasm -target wasm ."

while True:
    go_files = list(directory.rglob("*.go"))
    changed = False

    for f in go_files:
        try:
            mtime = f.stat().st_mtime
        except FileNotFoundError:
            continue

        if f not in go_mtimes or go_mtimes[f] != mtime:
            changed = True
            go_mtimes[f] = mtime

    if changed:
        time_str = datetime.datetime.fromtimestamp(time.time()).strftime('%H:%M:%S')
        print(f"{time_str}: Running {command}")
        subprocess.run(command, shell=True, cwd="_ABSOLUTE_CARROT")
        time_str = datetime.datetime.fromtimestamp(time.time()).strftime('%H:%M:%S')
        print(f"{time_str}: Finished {command}")

    index_path = "index.html"
    mtime = os.stat(index_path).st_mtime
    if index_mtime == None or index_mtime != mtime:
        index_mtime = mtime
        print(f"Running cp index.html 404.html")
        subprocess.run("cp index.html 404.html", shell=True)

    time.sleep(2)
