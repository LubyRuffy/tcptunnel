
call :build linux arm  
call :build linux amd64  
call :build linux 386  
call :build linux mips64le  
call :build linux mips64  
call :build darwin amd64  
call :build darwin 386   
call :build freebsd 386
call :build freebsd amd64
call :build windows 386 .exe
call :build windows amd64 .exe
goto :end

:build
  mkdir .\%1\%2
	set GOOS=%1
	set GOARCH=%2
	go build -o %1/%2/tcptunnel%3 -i -ldflags "-w -s" .
:end