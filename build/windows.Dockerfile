# Окружение для сборки Windows-версии из Linux.
# Fyne рисует окно через OpenGL и требует cgo, а cgo — компилятор C
# под целевую систему. mingw-w64 — это GCC, собирающий программы для Windows.
FROM golang:1.27
RUN apt-get update \
    && apt-get install -y --no-install-recommends gcc-mingw-w64-x86-64 \
    && rm -rf /var/lib/apt/lists/*
ENV CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc
