@echo off
set /p appname=ÇëÊäÈëappÃû³Æ: 

set GOOS=windows
set GOARCH=amd64
go build -o %appname%_%GOOS%_%GOARCH%.exe

set GOARCH=386
go build -o %appname%_%GOOS%_%GOARCH%.exe

set GOOS=linux
set GOARCH=amd64
go build -o %appname%_%GOOS%_%GOARCH%

set GOARCH=386
go build -o %appname%_%GOOS%_%GOARCH%

set GOARCH=arm
go build -o %appname%_%GOOS%_%GOARCH%

set GOARCH=arm64
go build -o %appname%_%GOOS%_%GOARCH%

set GOARCH=mips
go build -o %appname%_%GOOS%_%GOARCH%

set GOARCH=mipsle
go build -o %appname%_%GOOS%_%GOARCH%

set GOARCH=mips64
go build -o %appname%_%GOOS%_%GOARCH%

set GOARCH=mips64le
go build -o %appname%_%GOOS%_%GOARCH%

set GOOS=freebsd
set GOARCH=amd64
go build -o %appname%_%GOOS%_%GOARCH%

set GOARCH=386
go build -o %appname%_%GOOS%_%GOARCH%

set GOARCH=arm
go build -o %appname%_%GOOS%_%GOARCH%

set GOOS=darwin
set GOARCH=386
go build -o %appname%_%GOOS%_%GOARCH%

set GOARCH=amd64
go build -o %appname%_%GOOS%_%GOARCH%

set GOARCH=arm
go build -o %appname%_%GOOS%_%GOARCH%

set GOARCH=arm64
go build -o %appname%_%GOOS%_%GOARCH%

set GOOS=android
set GOARCH=arm
go build -o %appname%_%GOOS%_%GOARCH%

pause>nul